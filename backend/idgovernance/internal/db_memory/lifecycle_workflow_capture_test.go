package db_memory_test

import (
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/idgovernance/internal/db_memory"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/kernel"
)

//spec:covers REQ-SYSTEM-037: メモリのワークフロー確定も、条件を拒否したときは User と派生する実行を保存しない。
func TestConditionalUserWorkflowCaptureHasNoEffects(t *testing.T) {
	users := idmanagement.NewMemoryModule().UserRepo
	runs := memory.NewLifecycleWorkflowRunRepository()
	capture := &memory.UserWorkflowCapture{Users: users, Runs: runs}
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := &userdomain.User{ID: "00000000-0000-4000-8000-000000000605", TenantID: kernel.DefaultTenantID, PreferredUsername: "before", PasswordHash: "unused", CreatedAt: now, UpdatedAt: now}
	if err := users.Save(t.Context(), user); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{ID: "00000000-0000-4000-8000-000000000606", TenantID: user.TenantID, WorkflowID: "00000000-0000-4000-8000-000000000607", Revision: 1, SourceOccurrenceID: "edit", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: action, Outcome: igdomain.WorkflowStepPending}}
	updated := *user
	updated.PreferredUsername = "after"
	err := capture.SaveUserAndRuns(t.Context(), &updated, []*igdomain.WorkflowRun{run}, [][]igdomain.WorkflowStep{steps}, func(*userdomain.User) (bool, error) { return false, kernel.ErrPreconditionFailed })
	if !errors.Is(err, kernel.ErrPreconditionFailed) {
		t.Fatalf("refused capture error=%v", err)
	}
	if stored, err := runs.FindRun(t.Context(), user.TenantID, run.ID); err != nil || stored != nil {
		t.Fatalf("refused capture left run=%+v error=%v", stored, err)
	}
	if stored, err := users.FindBySub(t.Context(), user.ID); err != nil || stored.PreferredUsername != "before" {
		t.Fatalf("refused capture changed user=%+v error=%v", stored, err)
	}
}
