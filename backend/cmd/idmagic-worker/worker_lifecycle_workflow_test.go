package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idgovernance"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/db_memory"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igusecases "github.com/ambi/idmagic/backend/idgovernance/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// worker が組み立てる実行ハンドラーは、無効化されたワークフローの WorkflowRun を始めずに打ち切る。
// ワークフローの保存先を渡し忘れると、この打ち切りは黙って効かなくなる。
func TestWorkerLifecycleWorkflowHandlerCancelsRunsOfADisabledWorkflow(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	workflows := igmemory.NewLifecycleWorkflowRepository()
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now})
	if err := workflows.Save(ctx, &igdomain.LifecycleWorkflow{ID: "workflow-1", TenantID: "tenant-a", Name: "Leaver", Status: igdomain.LifecycleWorkflowDisabled, CurrentRevision: 1, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "occurrence-1", TargetUserID: "user-1", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	if _, err := runs.SaveRun(ctx, run, []igdomain.WorkflowStep{{RunID: run.ID, Action: action, Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: users, GroupRepo: groupmemory.NewGroupRepository()},
		IdGovernance: idgovernance.Module{LifecycleWorkflowRepo: workflows, LifecycleWorkflowRunRepo: runs},
		OAuth2:       oauth2.Module{EventSink: sinks_console.NewConsoleSink()},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}

	handler := igusecases.LifecycleWorkflowRunHandler(lifecycleWorkflowExecutorDeps(deps, logger))
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 5}); err != nil {
		t.Fatal(err)
	}

	if stored, err := runs.FindRun(ctx, "tenant-a", run.ID); err != nil || stored.Status != igdomain.WorkflowRunCanceled {
		t.Fatalf("run = %+v, %v; want canceled", stored, err)
	}
	if user, err := users.FindBySub(ctx, "user-1"); err != nil || user.Lifecycle.Status != idmdomain.UserStatusActive {
		t.Fatalf("user = %+v, %v; want the disable_user step never to run", user, err)
	}
}
