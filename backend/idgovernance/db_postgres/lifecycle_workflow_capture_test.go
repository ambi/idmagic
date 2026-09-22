package db_postgres_test

import (
	"context"
	"testing"

	igpostgres "github.com/ambi/idmagic/backend/idgovernance/db_postgres"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers EX-IDGOVERNANCE-004-01: User の変更と queued の WorkflowRun と pending のステップは 1 つのトランザクションで確定し、WorkflowRun の保存が失敗すれば User の変更も残らないこと、同じ重複排除キーの再配信は既存の WorkflowRun に収束してステップを作らないことを、PostgreSQL の保存先で固定する。
func TestUserWorkflowCaptureCommitsUserAndRunsTogether(t *testing.T) {
	pool := pgtest.Require(t)
	ctx := context.Background()
	tenant := pgfixtures.SeedTenant(t, pool)
	user := pgfixtures.SeedUser(t, pool, tenant.ID)
	now := pgfixtures.TestClock()
	workflows := &igpostgres.LifecycleWorkflowRepository{Pool: pool}
	workflow := &igdomain.LifecycleWorkflow{ID: pgfixtures.NewUUID(t), TenantID: tenant.ID, Name: "Joiner", Status: igdomain.LifecycleWorkflowDraft, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}
	if err := workflows.Save(ctx, workflow); err != nil {
		t.Fatal(err)
	}
	actions := []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}, {Kind: igdomain.WorkflowActionSendEmail, TemplateKey: "welcome"}}
	if err := workflows.SaveRevision(ctx, &igdomain.LifecycleWorkflowRevision{WorkflowID: workflow.ID, TenantID: tenant.ID, Revision: 1, Trigger: igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated}, Actions: actions, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	users := &userpg.UserRepository{Pool: pool}
	runs := &igpostgres.LifecycleWorkflowRunRepository{Pool: pool}
	capture := &igpostgres.UserWorkflowCapture{Pool: pool}
	planned := func(runID, workflowID string) (*igdomain.WorkflowRun, []igdomain.WorkflowStep) {
		run := &igdomain.WorkflowRun{ID: runID, TenantID: tenant.ID, WorkflowID: workflowID, Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: actions, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
		steps := []igdomain.WorkflowStep{
			{RunID: runID, Index: 0, Action: actions[0], Outcome: igdomain.WorkflowStepPending},
			{RunID: runID, Index: 1, Action: actions[1], Outcome: igdomain.WorkflowStepPending},
		}
		return run, steps
	}

	// 存在しないワークフローを指す WorkflowRun は外部キーで失敗する。User の変更も巻き戻らなければならない。
	renamed := *user
	renamed.PreferredUsername = user.PreferredUsername + "-renamed"
	broken, brokenSteps := planned(pgfixtures.NewUUID(t), pgfixtures.NewUUID(t))
	if err := capture.SaveUserAndRuns(ctx, &renamed, []*igdomain.WorkflowRun{broken}, [][]igdomain.WorkflowStep{brokenSteps}); err == nil {
		t.Fatal("a run the database refuses must fail the capture")
	}
	if stored, err := users.FindBySub(ctx, user.ID); err != nil || stored.PreferredUsername != user.PreferredUsername {
		t.Fatalf("user after the failed capture = %+v, %v; want the change rolled back", stored, err)
	}

	run, steps := planned(pgfixtures.NewUUID(t), workflow.ID)
	if err := capture.SaveUserAndRuns(ctx, &renamed, []*igdomain.WorkflowRun{run}, [][]igdomain.WorkflowStep{steps}); err != nil {
		t.Fatal(err)
	}
	if stored, err := users.FindBySub(ctx, user.ID); err != nil || stored.PreferredUsername != renamed.PreferredUsername {
		t.Fatalf("user after the capture = %+v, %v", stored, err)
	}
	stored, err := runs.FindRun(ctx, tenant.ID, run.ID)
	if err != nil || stored == nil || stored.Status != igdomain.WorkflowRunQueued || stored.JobID != nil {
		t.Fatalf("captured run = %+v, %v; want queued without a job", stored, err)
	}
	if persisted, err := runs.ListSteps(ctx, tenant.ID, run.ID); err != nil || len(persisted) != 2 || persisted[0].Outcome != igdomain.WorkflowStepPending || persisted[1].Outcome != igdomain.WorkflowStepPending {
		t.Fatalf("steps = %+v, %v; want two pending steps", persisted, err)
	}

	redelivered, redeliveredSteps := planned(pgfixtures.NewUUID(t), workflow.ID)
	if created, err := runs.SaveRun(ctx, redelivered, redeliveredSteps); err != nil || created {
		t.Fatalf("redelivered SaveRun = %v, %v; want it to converge", created, err)
	}
	if listed, err := runs.ListRuns(ctx, tenant.ID, workflow.ID, 10); err != nil || len(listed) != 1 {
		t.Fatalf("runs = %d, %v; want one", len(listed), err)
	}
	if persisted, err := runs.ListSteps(ctx, tenant.ID, redelivered.ID); err != nil || len(persisted) != 0 {
		t.Fatalf("redelivered steps = %d, %v; want none", len(persisted), err)
	}
}
