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
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 1, "admin", time.Time{}); !errors.Is(err, usecases.ErrWorkflowRevisionConflict) {
		t.Fatalf("stale enable error = %v", err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 2, "admin", time.Time{}); err != nil {
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

// wi-221: Create/Update/Enable/Disable がそれぞれ対応する audit event を発行する
// こと。以前は DeleteLifecycleWorkflow 以外の操作が一切 event を発行していなかった。
func TestLifecycleWorkflowCreateUpdateEnableDisableEmitAuditEvents(t *testing.T) {
	deps := usecases.LifecycleWorkflowDeps{Repo: idmmemory.NewLifecycleWorkflowRepository(), RunRepo: idmmemory.NewLifecycleWorkflowRunRepository()}
	var events []spec.DomainEvent
	deps.Emit = func(event spec.DomainEvent) error { events = append(events, event); return nil }

	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, usecases.CreateLifecycleWorkflowInput{Name: "Joiner", ActorUserID: "admin-1", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Now: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("CreateLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.UpdateLifecycleWorkflow(workflowContext(), deps, usecases.UpdateLifecycleWorkflowInput{WorkflowID: workflow.ID, ExpectedRevision: 1, Name: "Joiner v2", ActorUserID: "admin-1", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionEnableUser}}}); err != nil {
		t.Fatalf("UpdateLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 2, "admin-1", time.Time{}); err != nil {
		t.Fatalf("EnableLifecycleWorkflow: %v", err)
	}
	if _, err := usecases.DisableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 2, "admin-1", time.Time{}); err != nil {
		t.Fatalf("DisableLifecycleWorkflow: %v", err)
	}

	want := []string{"LifecycleWorkflowCreated", "LifecycleWorkflowUpdated", "LifecycleWorkflowEnabled", "LifecycleWorkflowDisabled"}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %d events", events, len(want))
	}
	for i, eventType := range want {
		if events[i].EventType() != eventType {
			t.Fatalf("events[%d] = %s, want %s", i, events[i].EventType(), eventType)
		}
	}
}

// wi-221: disable は未開始の queued run を cancel し、canceled になった run ごとに
// LifecycleWorkflowRunCanceled を発行する。
func TestDisableLifecycleWorkflowEmitsRunCanceledForQueuedRuns(t *testing.T) {
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	runRepo := idmmemory.NewLifecycleWorkflowRunRepository()
	deps := usecases.LifecycleWorkflowDeps{Repo: workflowRepo, RunRepo: runRepo}
	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, workflowInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 1, "admin-1", time.Time{}); err != nil {
		t.Fatal(err)
	}
	run := &idmdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: workflow.ID, Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: "user-1", TriggerKind: idmdomain.WorkflowTriggerUserCreated, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Status: idmdomain.WorkflowRunQueued, TriggeredAt: time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)}
	steps := []idmdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: run.Actions[0], Outcome: idmdomain.WorkflowStepPending}}
	if created, err := runRepo.SaveRun(workflowContext(), run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}

	var events []spec.DomainEvent
	deps.Emit = func(event spec.DomainEvent) error { events = append(events, event); return nil }
	if _, err := usecases.DisableLifecycleWorkflow(workflowContext(), deps, workflow.ID, 1, "admin-1", time.Time{}); err != nil {
		t.Fatalf("DisableLifecycleWorkflow: %v", err)
	}
	if len(events) != 2 || events[0].EventType() != "LifecycleWorkflowDisabled" || events[1].EventType() != "LifecycleWorkflowRunCanceled" {
		t.Fatalf("events = %#v, want [LifecycleWorkflowDisabled LifecycleWorkflowRunCanceled]", events)
	}
	canceledEvent, ok := events[1].(*spec.LifecycleWorkflowRunCanceled)
	if !ok || canceledEvent.RunID != run.ID || canceledEvent.TargetUserID != run.TargetUserID {
		t.Fatalf("canceled event = %#v", events[1])
	}
}

func TestLifecycleWorkflowTenantIsolation(t *testing.T) {
	deps := usecases.LifecycleWorkflowDeps{Repo: idmmemory.NewLifecycleWorkflowRepository()}
	workflow, err := usecases.CreateLifecycleWorkflow(workflowContext(), deps, workflowInput())
	if err != nil {
		t.Fatal(err)
	}
	other := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-b"}, "", "")
	if _, err := usecases.DisableLifecycleWorkflow(other, deps, workflow.ID, workflow.CurrentRevision, "admin", time.Time{}); !errors.Is(err, usecases.ErrLifecycleWorkflowNotFound) {
		t.Fatalf("cross-tenant access error = %v", err)
	}
}

// wi-222: dry-run must evaluate enabled_revision, not a later unenabled draft
// edit, so a draft change never appears to the admin as production behavior.
func TestDryRunLifecycleWorkflowUsesEnabledRevisionNotDraft(t *testing.T) {
	ctx := workflowContext()
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	users := idmmemory.NewUserRepository()
	deps := usecases.LifecycleWorkflowDeps{Repo: workflowRepo}
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	workflow, err := usecases.CreateLifecycleWorkflow(ctx, deps, usecases.CreateLifecycleWorkflowInput{Name: "Joiner", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionEnableUser}}, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(ctx, deps, workflow.ID, 1, "admin", now); err != nil {
		t.Fatal(err)
	}
	// Draft edit after enable: current_revision moves to 2 but enabled_revision stays 1.
	if _, err := usecases.UpdateLifecycleWorkflow(ctx, deps, usecases.UpdateLifecycleWorkflowInput{WorkflowID: workflow.ID, ExpectedRevision: 1, Name: "Joiner v2", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionDisableUser}}, Now: now}); err != nil {
		t.Fatal(err)
	}
	result, err := usecases.DryRunLifecycleWorkflow(ctx, usecases.DryRunLifecycleWorkflowDeps{Repo: workflowRepo, UserRepo: users}, workflow.ID, user.ID, now)
	if err != nil {
		t.Fatalf("DryRunLifecycleWorkflow: %v", err)
	}
	if result.Revision != 1 {
		t.Fatalf("revision = %d, want enabled_revision 1", result.Revision)
	}
	if len(result.Steps) != 1 || result.Steps[0].ActionKind != idmdomain.WorkflowActionEnableUser {
		t.Fatalf("steps = %#v, want the enabled revision's enable_user action, not the draft's disable_user", result.Steps)
	}
	if result.Steps[0].Outcome != idmdomain.WorkflowActionWouldChange {
		t.Fatalf("outcome = %s, want would_change for a disabled user", result.Steps[0].Outcome)
	}
}

// wi-222: an action the target User already satisfies must report no_op, not
// a hard-coded would_change.
func TestDryRunLifecycleWorkflowNoOpWhenUserAlreadyAtGoalState(t *testing.T) {
	ctx := workflowContext()
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	users := idmmemory.NewUserRepository()
	groups := idmmemory.NewGroupRepository()
	deps := usecases.LifecycleWorkflowDeps{Repo: workflowRepo}
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	group := &idmdomain.Group{ID: "group-1", TenantID: "tenant-a", Name: "Engineering", MembershipType: idmdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now}
	if err := groups.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	if ok, err := groups.AddMember(ctx, &idmdomain.GroupMember{GroupID: group.ID, UserID: user.ID, CreatedAt: now}); err != nil || !ok {
		t.Fatalf("seed AddMember = %v, %v", ok, err)
	}
	workflow, err := usecases.CreateLifecycleWorkflow(ctx, deps, usecases.CreateLifecycleWorkflowInput{Name: "Joiner", Trigger: idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated}, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionAddGroupMember, GroupID: group.ID}}, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	result, err := usecases.DryRunLifecycleWorkflow(ctx, usecases.DryRunLifecycleWorkflowDeps{Repo: workflowRepo, UserRepo: users, GroupRepo: groups}, workflow.ID, user.ID, now)
	if err != nil {
		t.Fatalf("DryRunLifecycleWorkflow: %v", err)
	}
	if len(result.Steps) != 1 || result.Steps[0].Outcome != idmdomain.WorkflowActionNoOp {
		t.Fatalf("steps = %#v, want a single no_op step", result.Steps)
	}
	stillMember, err := groups.ListGroupsByUser(ctx, "tenant-a", user.ID)
	if err != nil || len(stillMember) != 1 {
		t.Fatalf("dry-run must not mutate membership: %#v, %v", stillMember, err)
	}
}

// wi-222: when the trigger's filters don't match the target User's current
// attributes, dry-run must say so instead of pretending the actions would run.
func TestDryRunLifecycleWorkflowBlockedWhenTriggerFiltersDoNotMatch(t *testing.T) {
	ctx := workflowContext()
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	users := idmmemory.NewUserRepository()
	deps := usecases.LifecycleWorkflowDeps{Repo: workflowRepo}
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	department := "sales"
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]idmdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	trigger := idmdomain.WorkflowTrigger{Kind: idmdomain.WorkflowTriggerUserCreated, Filters: []idmdomain.WorkflowFilter{{Field: "department", Operator: idmdomain.WorkflowFilterEqual, Value: "engineering"}}}
	workflow, err := usecases.CreateLifecycleWorkflow(ctx, deps, usecases.CreateLifecycleWorkflowInput{Name: "Joiner", Trigger: trigger, Actions: []idmdomain.WorkflowAction{{Kind: idmdomain.WorkflowActionEnableUser}}, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	result, err := usecases.DryRunLifecycleWorkflow(ctx, usecases.DryRunLifecycleWorkflowDeps{Repo: workflowRepo, UserRepo: users}, workflow.ID, user.ID, now)
	if err != nil {
		t.Fatalf("DryRunLifecycleWorkflow: %v", err)
	}
	if len(result.Steps) != 1 || result.Steps[0].Outcome != idmdomain.WorkflowActionBlocked || result.Steps[0].Reason != "trigger_not_matched" {
		t.Fatalf("steps = %#v, want blocked/trigger_not_matched", result.Steps)
	}
}

func TestDryRunLifecycleWorkflowTargetUserNotFound(t *testing.T) {
	ctx := workflowContext()
	workflowRepo := idmmemory.NewLifecycleWorkflowRepository()
	users := idmmemory.NewUserRepository()
	deps := usecases.LifecycleWorkflowDeps{Repo: workflowRepo}
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	workflow, err := usecases.CreateLifecycleWorkflow(ctx, deps, workflowInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.DryRunLifecycleWorkflow(ctx, usecases.DryRunLifecycleWorkflowDeps{Repo: workflowRepo, UserRepo: users}, workflow.ID, "missing-user", now); !errors.Is(err, usecases.ErrLifecycleWorkflowTargetUserNotFound) {
		t.Fatalf("error = %v, want ErrLifecycleWorkflowTargetUserNotFound", err)
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
		if _, err := usecases.EnableLifecycleWorkflow(ctx, usecases.LifecycleWorkflowDeps{Repo: workflowRepo}, workflow.ID, 1, "admin", time.Time{}); err != nil {
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
