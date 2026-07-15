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
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func workflowContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
}

func workflowInput() usecases.CreateLifecycleWorkflowInput {
	return usecases.CreateLifecycleWorkflowInput{Name: "Joiner", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)}
}

type orderedWorkflowRepository struct {
	workflows map[string]*idmdomain.LifecycleWorkflow
	order     []string
}

func (r *orderedWorkflowRepository) List(_ context.Context, tenantID string) ([]*idmdomain.LifecycleWorkflow, error) {
	workflows := make([]*idmdomain.LifecycleWorkflow, 0, len(r.workflows))
	for _, workflow := range r.workflows {
		if workflow.TenantID == tenantID {
			workflows = append(workflows, workflow)
		}
	}
	return workflows, nil
}

func (r *orderedWorkflowRepository) Find(_ context.Context, tenantID, workflowID string) (*idmdomain.LifecycleWorkflow, error) {
	workflow := r.workflows[workflowID]
	if workflow == nil || workflow.TenantID != tenantID {
		return nil, errors.New("workflow not found")
	}
	return workflow, nil
}

func (r *orderedWorkflowRepository) Save(_ context.Context, workflow *idmdomain.LifecycleWorkflow) error {
	r.order = append(r.order, "workflow")
	r.workflows[workflow.ID] = workflow
	return nil
}

func (r *orderedWorkflowRepository) FindRevision(_ context.Context, _, _ string, _ int64) (*idmdomain.LifecycleWorkflowRevision, error) {
	return nil, errors.New("workflow revision not found")
}

func (r *orderedWorkflowRepository) SaveRevision(_ context.Context, revision *idmdomain.LifecycleWorkflowRevision) error {
	r.order = append(r.order, "revision")
	if r.workflows[revision.WorkflowID] == nil {
		return errors.New("workflow must be stored before its revision")
	}
	return nil
}

func TestCreateLifecycleWorkflowStoresDefinitionBeforeRevision(t *testing.T) {
	repo := &orderedWorkflowRepository{workflows: map[string]*idmdomain.LifecycleWorkflow{}}
	if _, err := usecases.CreateLifecycleWorkflow(workflowContext(), usecases.LifecycleWorkflowDeps{Repo: repo}, workflowInput()); err != nil {
		t.Fatalf("CreateLifecycleWorkflow: %v", err)
	}
	if len(repo.order) != 2 || repo.order[0] != "workflow" || repo.order[1] != "revision" {
		t.Fatalf("save order = %v, want [workflow revision]", repo.order)
	}
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
	var emitted spec.DomainEvent
	deps.Emit = func(event spec.DomainEvent) error { emitted = event; return nil }
	if err := usecases.DeleteLifecycleWorkflow(workflowContext(), deps, workflow.ID, updated.CurrentRevision, "admin", time.Time{}); err != nil {
		t.Fatalf("DeleteLifecycleWorkflow: %v", err)
	}
	if emitted == nil || emitted.EventType() != "LifecycleWorkflowDeleted" {
		t.Fatalf("delete event = %#v", emitted)
	}
	if _, err := usecases.UpdateLifecycleWorkflow(workflowContext(), deps, usecases.UpdateLifecycleWorkflowInput{WorkflowID: workflow.ID}); !errors.Is(err, usecases.ErrLifecycleWorkflowNotFound) {
		t.Fatalf("update deleted workflow error = %v", err)
	}
	if _, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, usecases.CreateLifecycleWorkflowInput{Name: "Joiner v2", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}}); err != nil {
		t.Fatalf("reuse deleted workflow name: %v", err)
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
