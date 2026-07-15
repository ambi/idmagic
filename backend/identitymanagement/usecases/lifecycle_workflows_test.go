package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authnmemory "github.com/ambi/idmagic/backend/authentication/adapters/persistence/memory"
	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func workflowContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
}

func workflowInput() usecases.CreateLifecycleWorkflowInput {
	return usecases.CreateLifecycleWorkflowInput{Name: "Joiner", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)}
}

func TestLifecycleWorkflowCreateUpdateAndTransitions(t *testing.T) {
	deps := usecases.LifecycleWorkflowDeps{Repo: idmmemory.NewLifecycleWorkflowRepository()}
	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, workflowInput())
	if err != nil {
		t.Fatalf("CreateLifecycleWorkflow: %v", err)
	}
	updated, err := usecases.UpdateLifecycleWorkflow(workflowContext(), deps, usecases.UpdateLifecycleWorkflowInput{WorkflowID: workflow.ID, ExpectedRevision: 1, Name: "Joiner v2", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionEnableUser}}})
	if err != nil || updated.CurrentRevision != 2 {
		t.Fatalf("UpdateLifecycleWorkflow = %#v, %v", updated, err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 1, time.Time{}); !errors.Is(err, usecases.ErrWorkflowRevisionConflict) {
		t.Fatalf("stale enable error = %v", err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 2, time.Time{}); err != nil {
		t.Fatalf("EnableLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.DisableLifecycleWorkflow(workflowContext(), deps, workflow.ID, updated.CurrentRevision, time.Time{}); err != nil {
		t.Fatalf("DisableLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.ArchiveLifecycleWorkflow(workflowContext(), deps, workflow.ID, updated.CurrentRevision, time.Time{}); err != nil {
		t.Fatalf("ArchiveLifecycleWorkflow: %v", err)
	}
}

func TestLifecycleWorkflowTenantIsolation(t *testing.T) {
	deps := usecases.LifecycleWorkflowDeps{Repo: idmmemory.NewLifecycleWorkflowRepository()}
	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, workflowInput())
	if err != nil {
		t.Fatal(err)
	}
	other := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	if _, err := usecases.DisableLifecycleWorkflow(other, deps, workflow.ID, workflow.CurrentRevision, time.Time{}); !errors.Is(err, usecases.ErrLifecycleWorkflowNotFound) {
		t.Fatalf("cross-tenant access error = %v", err)
	}
}

func TestUserMutationsCaptureMatchingWorkflowRuns(t *testing.T) {
	ctx := workflowContext()
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	active, disabled := idmdomain.UserStatusActive, idmdomain.UserStatusDisabled
	for _, trigger := range []idmdomain.WorkflowTrigger{
		{Kind: idmdomain.WorkflowTriggerUserCreated},
		{Kind: idmdomain.WorkflowTriggerUserAttributesChanged, WatchedAttributes: []string{"department"}},
		{Kind: idmdomain.WorkflowTriggerUserStatusChanged, FromStatus: &active, ToStatus: &disabled},
	} {
		workflow, err := usecases.CreateLifecycleWorkflow(ctx, usecases.LifecycleWorkflowDeps{Repo: workflowRepo}, usecases.CreateLifecycleWorkflowInput{Name: string(trigger.Kind), Trigger: trigger, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := usecases.EnableLifecycleWorkflow(ctx, usecases.LifecycleWorkflowDeps{Repo: workflowRepo}, workflow.ID, 1, time.Time{}); err != nil {
			t.Fatal(err)
		}
	}
	users := idmmemory.NewUserRepository()
	runs := idmmemory.NewLifecycleWorkflowRunRepository()
	deps := usecases.AdminUserDeps{
		UserRepo: users, WorkflowRepo: workflowRepo, WorkflowRunRepo: runs,
		WorkflowCapture: &idmmemory.UserWorkflowCapture{Users: users, Runs: runs},
		PasswordHasher:  crypto.NewArgon2idPasswordHasher(), PasswordHistoryRepo: authnmemory.NewPasswordHistoryRepository(),
	}
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	user, err := usecases.CreateUser(ctx, deps, usecases.CreateUserInput{PreferredUsername: "alice", Password: "initial-password-9182", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	department := "Engineering"
	if _, err := usecases.UpdateUser(ctx, deps, usecases.UpdateUserInput{Sub: user.ID, Attributes: &map[string]idmdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}, Now: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.SetUserDisabled(ctx, deps, "operator", user.ID, true, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	captured, err := runs.ListUnenqueuedRuns(ctx, 10)
	if err != nil || len(captured) != 3 {
		t.Fatalf("captured runs = %d, %v", len(captured), err)
	}
}
