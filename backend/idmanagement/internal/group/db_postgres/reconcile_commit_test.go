package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/audit"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/jackc/pgx/v5"
)

//spec:covers REQ-JOBS-002
//spec:covers REQ-JOBS-007
func TestGroupImportReconciliationEmitsAfterCommitAndDeduplicates(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	mutation := reconcileMutation(t, db)
	quotas := tenancy.NewQuotaRepository(db)
	limit := 1
	if err := quotas.SetQuota(ctx, mutation.After.TenantID, &tenancydomain.TenantQuota{ActiveJobs: &limit}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	committer := NewGroupImportCommitter(db, tenancy.QuotaRepositoryInTx, jobs.NewEnqueuerInTx, func(event spec.DomainEvent) {
		assertGroupStored(ctx, t, db, mutation.After.TenantID, mutation.After.ID, true)
		enqueued, ok := event.(*jobsdomain.JobEnqueued)
		if !ok {
			t.Fatalf("event = %T, want JobEnqueued", event)
		}
		stored, err := jobs.NewPostgresModule(db).Repo.Get(ctx, enqueued.JobID)
		if err != nil || stored == nil {
			t.Fatalf("job not committed before event: job=%+v error=%v", stored, err)
		}
		var params map[string]any
		if err := json.Unmarshal(stored.Params, &params); err != nil {
			t.Fatal(err)
		}
		if enqueued.TenantID != mutation.After.TenantID || enqueued.Kind != jobsdomain.KindDynamicGroupReconcile ||
			stored.Lane != jobsdomain.LaneBulk || stored.MaxAttempts != jobsdomain.DefaultMaxAttempts ||
			stored.Status != jobsdomain.StatusQueued || !stored.RunAt.Equal(mutation.Now) ||
			params["group_id"] != mutation.After.ID || params["rule_version"] != float64(1) {
			t.Fatalf("event=%+v job=%+v params=%+v", enqueued, stored, params)
		}
		events = append(events, event)
	}, audit.RecordWriterInTx)
	if err := committer.CommitGroupImportRow(ctx, mutation); err != nil {
		t.Fatal(err)
	}
	// 同じ版の更新を繰り返しても、満杯の active_jobs で拒否しない。
	mutation.ConsumesGroupQuota = false
	if err := committer.CommitGroupImportRow(ctx, mutation); err != nil {
		t.Fatalf("deduplicated row: %v", err)
	}
	usage, err := quotas.GetUsage(ctx, mutation.After.TenantID)
	if err != nil || usage.ActiveJobs != 1 || usage.Groups != 1 || len(events) != 1 {
		t.Fatalf("usage=%+v error=%v events=%d, want one group, job and event", usage, err, len(events))
	}
}

//spec:covers REQ-IDMANAGEMENT-026
//spec:covers REQ-JOBS-002
func TestGroupImportReconciliationRollsBackOnCommitFailure(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	mutation := reconcileMutation(t, db)
	var events []spec.DomainEvent
	committer := NewGroupImportCommitter(commitFailureDB{DB: db}, tenancy.QuotaRepositoryInTx, jobs.NewEnqueuerInTx, func(event spec.DomainEvent) {
		events = append(events, event)
	}, audit.RecordWriterInTx)
	err := committer.CommitGroupImportRow(ctx, mutation)
	if !errors.Is(err, errReconcileCommit) {
		t.Fatalf("error=%v, want injected commit failure", err)
	}
	assertGroupStored(ctx, t, db, mutation.After.TenantID, mutation.After.ID, false)
	usage, err := tenancy.NewQuotaRepository(db).GetUsage(ctx, mutation.After.TenantID)
	if err != nil || usage.ActiveJobs != 0 || usage.Groups != 0 {
		t.Fatalf("usage=%+v error=%v, want no committed usage", usage, err)
	}
	if len(events) != 0 || len(importAuditEvents(ctx, t, mutation.After.TenantID)) != 0 {
		t.Fatalf("events=%+v, want no events or audit", events)
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE tenant_id = $1`, mutation.After.TenantID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("jobs=%d error=%v, want none", count, err)
	}
	assertNoReconcileRule(t, db, mutation.After.ID)
}

func reconcileMutation(t *testing.T, db sharedpg.DB) groupports.GroupImportRowMutation {
	t.Helper()
	tenant := seedTenant(t, db)
	actor := seedUser(t, db, tenant.ID)
	now := testClock()
	group := &groupdomain.Group{ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("dynamic-row"), CreatedAt: now, UpdatedAt: now}
	return groupports.GroupImportRowMutation{
		After: group, ActorUserID: actor.ID, AuditEventType: "group.imported", ConsumesGroupQuota: true,
		Rule:             &groupdomain.DynamicGroupRule{GroupID: group.ID, TenantID: tenant.ID, Expression: "true", Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now},
		ReconcileGroupID: group.ID, ReconcileVersion: 1, Now: now,
	}
}

func assertNoReconcileRule(t *testing.T, db sharedpg.DB, groupID string) {
	t.Helper()
	var count int
	if err := db.QueryRow(t.Context(), `SELECT count(*) FROM dynamic_group_rules WHERE group_id = $1`, groupID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("rules=%d error=%v, want none", count, err)
	}
}

var errReconcileCommit = errors.New("injected commit failure")

type commitFailureDB struct{ sharedpg.DB }

func (db commitFailureDB) Begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := db.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return commitFailureTx{Tx: tx}, nil
}

type commitFailureTx struct{ pgx.Tx }

func (commitFailureTx) Commit(context.Context) error { return errReconcileCommit }

func TestGroupImportCommitterRejectsMissingDependencies(t *testing.T) {
	db := pgtest.Require(t)
	for _, missing := range []string{"database", "quotas", "jobs", "events", "audit"} {
		t.Run(missing, func(t *testing.T) {
			var pool sharedpg.DB = db
			quotas := tenancy.QuotaRepositoryInTx
			enqueuers := jobs.NewEnqueuerInTx
			emit := func(spec.DomainEvent) {}
			auditInTx := audit.RecordWriterInTx
			switch missing {
			case "database":
				pool = nil
			case "quotas":
				quotas = nil
			case "jobs":
				enqueuers = nil
			case "events":
				emit = nil
			case "audit":
				auditInTx = nil
			}
			defer func() {
				if recover() == nil {
					t.Fatal("missing required dependency did not panic")
				}
			}()
			NewGroupImportCommitter(pool, quotas, enqueuers, emit, auditInTx)
		})
	}
}
