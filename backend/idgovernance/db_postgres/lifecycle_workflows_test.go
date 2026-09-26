package db_postgres_test

import (
	"context"
	"reflect"
	"testing"

	igpostgres "github.com/ambi/idmagic/backend/idgovernance/db_postgres"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestLifecycleWorkflowRepositoryReadsBackWhatItSaved(t *testing.T) {
	pool := pgtest.Require(t)
	ctx := context.Background()
	tenant := pgfixtures.SeedTenant(t, pool)
	other := pgfixtures.SeedTenant(t, pool)
	now := pgfixtures.TestClock()
	repo := &igpostgres.LifecycleWorkflowRepository{Pool: pool}
	description := "offboarding"
	enabled := int64(1)
	leaver := &igdomain.LifecycleWorkflow{
		ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Name: "Leaver", Description: &description,
		Status: igdomain.LifecycleWorkflowEnabled, CurrentRevision: 1, EnabledRevision: &enabled, CreatedAt: now, UpdatedAt: now,
	}
	joiner := &igdomain.LifecycleWorkflow{
		ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Name: "Joiner", Status: igdomain.LifecycleWorkflowDraft,
		CurrentRevision: 1, CreatedAt: now, UpdatedAt: now,
	}
	for _, workflow := range []*igdomain.LifecycleWorkflow{leaver, joiner} {
		if err := repo.Save(ctx, workflow); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	listed, err := repo.List(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	inUTC(listed...)
	if want := []*igdomain.LifecycleWorkflow{joiner, leaver}; !reflect.DeepEqual(listed, want) {
		t.Fatalf("List = %+v, want %+v ordered by name", derefWorkflows(listed), derefWorkflows(want))
	}
	found, err := repo.Find(ctx, tenant.ID, leaver.ID)
	inUTC(found)
	if err != nil || !reflect.DeepEqual(found, leaver) {
		t.Fatalf("Find = %+v, %v; want %+v", found, err, leaver)
	}
	if found, err := repo.Find(ctx, other.ID, leaver.ID); err != nil || found != nil {
		t.Fatalf("Find from another tenant = %+v, %v; want nil", found, err)
	}

	revision := &igdomain.LifecycleWorkflowRevision{
		WorkflowID: leaver.ID, TenantID: tenant.ID, Revision: 1,
		Trigger:   igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		Actions:   []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}},
		CreatedAt: now,
	}
	if err := repo.SaveRevision(ctx, revision); err != nil {
		t.Fatalf("SaveRevision: %v", err)
	}
	gotRevision, err := repo.FindRevision(ctx, tenant.ID, leaver.ID, 1)
	if gotRevision != nil {
		gotRevision.CreatedAt = gotRevision.CreatedAt.UTC()
	}
	if err != nil || !reflect.DeepEqual(gotRevision, revision) {
		t.Fatalf("FindRevision = %+v, %v; want %+v", gotRevision, err, revision)
	}
	if missing, err := repo.FindRevision(ctx, tenant.ID, leaver.ID, 2); err != nil || missing != nil {
		t.Fatalf("FindRevision of an absent revision = %+v, %v; want nil", missing, err)
	}
}

// inUTC は読み戻した時刻を保存時と同じ UTC にそろえる。PostgreSQL は時刻の瞬間だけを保ち、
// 読み取りは接続のタイムゾーンで返すので、所在地の違いは比べる対象から外す。
func inUTC(workflows ...*igdomain.LifecycleWorkflow) {
	for _, workflow := range workflows {
		if workflow != nil {
			workflow.CreatedAt, workflow.UpdatedAt = workflow.CreatedAt.UTC(), workflow.UpdatedAt.UTC()
		}
	}
}

func derefWorkflows(workflows []*igdomain.LifecycleWorkflow) []igdomain.LifecycleWorkflow {
	out := make([]igdomain.LifecycleWorkflow, len(workflows))
	for i, workflow := range workflows {
		out[i] = *workflow
	}
	return out
}
