package idmanagement_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/idmanagement"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

type importHandlers map[jobsdomain.JobKind]jobsports.Handler

func (h importHandlers) Register(kind jobsdomain.JobKind, handler jobsports.Handler) {
	h[kind] = handler
}

func TestJobHandlersRequireAnEventSink(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("missing event sink did not panic")
		}
	}()
	idmanagement.NewMemoryModule().RegisterJobHandlers(importHandlers{}, idmanagement.JobHandlerDeps{})
}

//spec:covers REQ-JOBS-002
//spec:covers REQ-IDMANAGEMENT-026
func TestPostgresGroupImportHandlerWiresTransactionalReconciliation(t *testing.T) {
	db := pgtest.Require(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	tenantID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	tenant := &tenancydomain.Tenant{ID: tenantID, Realm: "csv-" + tenantID, DisplayName: "CSV", Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	if err := tenancy.NewPostgresModule(db).TenantRepo.Save(t.Context(), tenant); err != nil {
		t.Fatal(err)
	}
	ctx := tenantports.WithTenant(t.Context(), tenant, "", "")
	idm := idmanagement.NewPostgresModule(db).WithPostgresImports(db, idmanagement.PostgresImportDeps{
		QuotasInTx:  tenancy.QuotaRepositoryInTx,
		AuditInTx:   audit.RecordWriterInTx,
		HistoryInTx: authentication.PasswordHistoryWriterInTx,
	})
	actorID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	if err := idm.UserRepo.Save(ctx, &userdomain.User{ID: actorID, TenantID: tenant.ID, PreferredUsername: "operator", PasswordHash: "x", Roles: []string{}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	repo := jobs.NewPostgresModule(db).Repo
	var events []spec.DomainEvent
	handlers := importHandlers{}
	idm.RegisterJobHandlers(handlers, idmanagement.JobHandlerDeps{
		Jobs: repo, EnqueuerInTx: jobs.NewEnqueuerInTx,
		AttrSchemaRepo: idm.AttrSchemaRepo, GroupAttrSchemaRepo: idm.GroupAttrSchemaRepo,
		GroupOwnershipGuard: sourcing.NewPostgresModule(db).GroupOwnershipGuard(),
		Emit:                func(event spec.DomainEvent) error { events = append(events, event); return nil },
	})
	start := groupusecases.GroupImportStartDeps{
		Artifacts: idm.CSVArtifacts, Jobs: repo,
		Enqueuer: jobs.NewEnqueuer(repo, tenancy.NewQuotaRepository(db), func(spec.DomainEvent) {}),
	}
	preview, err := groupusecases.StartGroupImportPreview(ctx, start, actorID,
		strings.NewReader("name,membership_type,dynamic_rule_expression,dynamic_rule_enabled\nimported,dynamic,user.email_verified == true,true\n"), now)
	if err != nil {
		t.Fatal(err)
	}
	result, err := handlers[jobsdomain.KindGroupImportPreview](ctx, preview)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := repo.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, now)
	if err != nil || len(claimed) != 1 || claimed[0].ID != preview.ID {
		t.Fatalf("claimed=%+v error=%v, want preview", claimed, err)
	}
	if _, err := repo.Complete(ctx, preview.ID, "worker", result, now); err != nil {
		t.Fatal(err)
	}
	// Runner の代わりに直接 Complete したので、終端の使用量の解放もここで行う。
	if err := tenancy.NewQuotaRepository(db).Decrement(ctx, tenant.ID, tenancydomain.ResourceActiveJobs, 1); err != nil {
		t.Fatal(err)
	}
	apply, err := groupusecases.StartGroupImportApply(ctx, start, actorID, preview.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := handlers[jobsdomain.KindGroupImportApply](ctx, apply)
	if err != nil {
		t.Fatal(err)
	}
	var applied groupusecases.GroupImportResult
	if err := json.Unmarshal(raw, &applied); err != nil || applied.CreatedRows != 1 || applied.RejectedRows != 0 {
		rows, readErr := groupusecases.ReadGroupImportErrorRange(ctx, idm.CSVArtifacts, tenant.ID, applied, 1, 10)
		t.Fatalf("result=%s error=%v rows=%+v readError=%v, want one accepted row", raw, err, rows, readErr)
	}
	if len(events) != 1 {
		t.Fatalf("events=%+v, want one reconciliation event", events)
	}
	enqueued, ok := events[0].(*jobsdomain.JobEnqueued)
	if !ok || enqueued.Kind != jobsdomain.KindDynamicGroupReconcile || enqueued.TenantID != tenant.ID {
		t.Fatalf("event=%+v, want JobEnqueued for reconciliation", events[0])
	}
	job, err := repo.Get(ctx, enqueued.JobID)
	if err != nil || job == nil || job.Status != jobsdomain.StatusQueued {
		t.Fatalf("job=%+v error=%v, want committed queued reconciliation", job, err)
	}
	records, err := audit.NewPostgresModule(db).AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: tenant.ID, Type: "GroupCreated", UserID: actorID})
	if err != nil || len(records) != 1 || records[0].Payload["actorUserId"] != actorID {
		t.Fatalf("audit=%+v error=%v, want one committed group creation indexed by the actor", records, err)
	}
	usage, err := tenancy.NewQuotaRepository(db).GetUsage(ctx, tenant.ID)
	if err != nil || usage.ActiveJobs != 2 || usage.Groups != 1 {
		t.Fatalf("usage=%+v error=%v, want group and two active jobs", usage, err)
	}
}
