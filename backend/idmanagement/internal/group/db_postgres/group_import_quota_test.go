package db_postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/audit"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type unmanagedImportGroups struct{}

func (unmanagedImportGroups) SourceManagedGroupIDs(context.Context, string, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// 一行目は再評価ジョブの active_jobs で、四行目は groups で拒否される。committer が
// 発行する active_jobs の拒否を use case が重ねず、groups の拒否を use case が補うので、
// 発行は拒否した行ごとにちょうど一件になる。
//
//spec:covers REQ-TENANCY-013: PostgreSQL の確定で CSV の行を groups または active_jobs の Hard Quota で拒否したとき、拒否した行ごとに、その資源の QuotaExceeded をちょうど一件発行すること。
func TestApplyGroupImportEmitsOneQuotaExceededPerRefusedRowInPostgres(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	tenant := seedTenant(t, db)
	actor := seedUser(t, db, tenant.ID)
	quotas := tenancy.NewQuotaRepository(db)
	groupLimit, jobLimit := 2, 0
	if err := quotas.SetQuota(ctx, tenant.ID, &tenancydomain.TenantQuota{Groups: &groupLimit, ActiveJobs: &jobLimit}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	emit := func(event spec.DomainEvent) { events = append(events, event) }
	plan := groupusecases.GroupImportPlanDeps{GroupRepo: &GroupRepository{Pool: db}, OwnershipGuard: unmanagedImportGroups{}}
	deps := groupusecases.GroupImportApplyDeps{
		Plan:      plan,
		Committer: NewGroupImportCommitter(db, tenancy.QuotaRepositoryInTx, jobs.NewEnqueuerInTx, emit, audit.RecordWriterInTx),
		Emit:      func(event spec.DomainEvent) error { emit(event); return nil },
	}
	csv := "name,membership_type,dynamic_rule_expression,dynamic_rule_enabled\n" +
		"dyn,dynamic,user.preferred_username.size() > 0,true\n" +
		"alpha,manual,,\n" +
		"beta,manual,,\n" +
		"gamma,manual,,\n"

	var rows []groupdomain.GroupImportRowPlan
	summary, err := groupusecases.ApplyGroupImport(tenantports.WithTenant(ctx, tenant, "", ""), deps, strings.NewReader(csv),
		idmdomain.DefaultCSVTransferPolicy(), actor.ID, testClock(), func(row groupdomain.GroupImportRowPlan) error {
			rows = append(rows, row)
			return nil
		})
	if err != nil {
		t.Fatal(err)
	}
	if summary.CreatedRows != 2 || summary.RejectedRows != 2 {
		t.Fatalf("summary=%+v rows=%+v", summary, rows)
	}
	assertGroupUsage(ctx, t, quotas, tenant.ID, 2)
	want := []string{tenancydomain.ResourceActiveJobs, tenancydomain.ResourceGroups}
	if len(events) != len(want) {
		t.Fatalf("events=%+v, want one refusal for each of %v", events, want)
	}
	for i, event := range events {
		refused, ok := event.(*tenancydomain.QuotaExceeded)
		if !ok || refused.TenantID != tenant.ID || refused.Resource != want[i] || !refused.HardLimit {
			t.Fatalf("events[%d]=%+v, want a %s Hard Quota refusal", i, event, want[i])
		}
	}
}
