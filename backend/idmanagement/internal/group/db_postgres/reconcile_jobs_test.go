package db_postgres

import (
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/shared/spec"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-TENANCY-013
//spec:covers REQ-IDMANAGEMENT-026
func TestGroupImportReconciliationRejectsAnActiveJobQuota(t *testing.T) {
	db := pgtest.Require(t)
	ctx := t.Context()
	tenant := seedTenant(t, db)
	actor := seedUser(t, db, tenant.ID)
	quotas := tenancy.NewQuotaRepository(db)
	limit := 0
	if err := quotas.SetQuota(ctx, tenant.ID, &tenancydomain.TenantQuota{ActiveJobs: &limit}); err != nil {
		t.Fatal(err)
	}
	now := testClock()
	group := &groupdomain.Group{ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("quota-row"), CreatedAt: now, UpdatedAt: now}
	mutation := groupports.GroupImportRowMutation{
		After: group, ActorUserID: actor.ID, AuditEventType: "group.imported", ConsumesGroupQuota: true,
		ReconcileGroupID: group.ID, ReconcileVersion: 1, Now: now,
	}
	mutation.Rule = &groupdomain.DynamicGroupRule{GroupID: group.ID, TenantID: tenant.ID, Expression: "true", Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now}
	var events []spec.DomainEvent
	err := NewGroupImportRowCommitter(db, tenancy.QuotaRepositoryInTx, jobs.NewEnqueuerInTx, func(event spec.DomainEvent) {
		assertGroupUsage(ctx, t, quotas, tenant.ID, 0)
		events = append(events, event)
	}).CommitGroupImportRow(ctx, mutation)
	var exceeded *tenancydomain.QuotaExceededError
	if !errors.As(err, &exceeded) || exceeded.Resource != tenancydomain.ResourceActiveJobs {
		t.Fatalf("CommitGroupImportRow error = %v, want active_jobs quota refusal", err)
	}
	assertGroupStored(ctx, t, db, tenant.ID, group.ID, false)
	assertGroupUsage(ctx, t, quotas, tenant.ID, 0)
	usage, err := quotas.GetUsage(ctx, tenant.ID)
	if err != nil || usage.ActiveJobs != 0 {
		t.Fatalf("usage = %+v, error = %v, want no active jobs", usage, err)
	}
	if events := importAuditEvents(ctx, t, tenant.ID); len(events) != 0 {
		t.Fatalf("audit events = %+v, want none", events)
	}
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM jobs WHERE tenant_id = $1`, tenant.ID).Scan(&count); err != nil || count != 0 {
		t.Fatalf("job count = %d, error = %v, want zero", count, err)
	}
	assertNoReconcileRule(t, db, group.ID)
	if len(events) != 1 {
		t.Fatalf("events = %+v, want one quota refusal", events)
	}
	refused, ok := events[0].(*tenancydomain.QuotaExceeded)
	if !ok || refused.TenantID != tenant.ID || refused.Resource != tenancydomain.ResourceActiveJobs || !refused.HardLimit {
		t.Fatalf("event = %+v, want active_jobs quota refusal", events[0])
	}
}
