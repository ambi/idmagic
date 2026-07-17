package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/adapters/persistence/memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/adapters/persistence/memory"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	"github.com/ambi/idmagic/backend/idgovernance/usecases"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/adapters/persistence/memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type failOnceJobRepository struct {
	jobsports.JobRepository
	fail bool
}

func (r *failOnceJobRepository) Enqueue(ctx context.Context, in jobsports.EnqueueInput) (*jobsdomain.Job, bool, error) {
	if r.fail {
		r.fail = false
		return nil, false, errors.New("transient enqueue failure")
	}
	return r.JobRepository.Enqueue(ctx, in)
}

func TestDispatchQueuedLifecycleWorkflowRunsAttachesDeduplicatedJob(t *testing.T) {
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: "user-1", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(context.Background(), run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	jobs := &failOnceJobRepository{JobRepository: jobsmemory.NewJobRepository(), fail: true}
	deps := usecases.LifecycleWorkflowDispatcherDeps{RunRepo: runs, JobRepo: jobs}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(context.Background(), deps, 10, now); err == nil {
		t.Fatal("first dispatch must expose enqueue failure")
	}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(context.Background(), deps, 10, now); err != nil {
		t.Fatal(err)
	}
	stored, err := runs.FindRun(context.Background(), "tenant-a", run.ID)
	if err != nil || stored.JobID == nil {
		t.Fatalf("run job attachment = %#v, %v", stored, err)
	}
}

func TestLifecycleWorkflowRunHandlerCheckpointsAndSkipsCompletedStepsOnRetry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := idmmemory.NewUserRepository()
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	job := &jobsdomain.Job{TenantID: run.TenantID, Params: params}
	if _, err := handler(ctx, job); err != nil {
		t.Fatal(err)
	}
	stored, err := users.FindBySub(ctx, user.ID)
	if err != nil || stored.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("user status = %#v, %v", stored, err)
	}
	storedSteps, err := runs.ListSteps(ctx, run.TenantID, run.ID)
	if err != nil || storedSteps[0].Outcome != igdomain.WorkflowStepChanged {
		t.Fatalf("steps = %#v, %v", storedSteps, err)
	}
	if _, err := handler(ctx, job); err == nil {
		t.Fatal("terminal run must not execute again")
	}
}

// wi-221: a run whose only step succeeds emits RunStarted then RunSucceeded.
func TestLifecycleWorkflowRunHandlerEmitsRunStartedAndRunSucceeded(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := idmmemory.NewUserRepository()
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	var events []spec.DomainEvent
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users, Emit: func(e spec.DomainEvent) error { events = append(events, e); return nil }})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params}); err != nil {
		t.Fatal(err)
	}
	want := []string{"LifecycleWorkflowRunStarted", "LifecycleWorkflowRunSucceeded"}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %v", events, want)
	}
	for i, eventType := range want {
		if events[i].EventType() != eventType {
			t.Fatalf("events[%d] = %s, want %s", i, events[i].EventType(), eventType)
		}
	}
}

// wi-221: a run where every step fails must terminate as WorkflowRunFailed (not
// PartiallyFailed) and emit StepFailed followed by RunFailed. Before this fix
// the handler only ever distinguished succeeded/partially_failed, so a run with
// zero successful steps was misclassified as partially_failed.
func TestLifecycleWorkflowRunHandlerAllStepsFailedEmitsRunFailed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := idmmemory.NewUserRepository()
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "group-1"}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: action, Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	// GroupRepo left nil so the step fails with dependency_unavailable.
	var events []spec.DomainEvent
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users, Emit: func(e spec.DomainEvent) error { events = append(events, e); return nil }})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params}); err != nil {
		t.Fatal(err)
	}
	stored, err := runs.FindRun(ctx, run.TenantID, run.ID)
	if err != nil || stored.Status != igdomain.WorkflowRunFailed {
		t.Fatalf("run status = %#v, %v, want failed", stored, err)
	}
	want := []string{"LifecycleWorkflowRunStarted", "LifecycleWorkflowStepFailed", "LifecycleWorkflowRunFailed"}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %v", events, want)
	}
	for i, eventType := range want {
		if events[i].EventType() != eventType {
			t.Fatalf("events[%d] = %s, want %s", i, events[i].EventType(), eventType)
		}
	}
}

// wi-221: a run with one failing step and one succeeding step terminates as
// WorkflowRunPartiallyFailed.
func TestLifecycleWorkflowRunHandlerMixedOutcomeEmitsRunPartiallyFailed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := idmmemory.NewUserRepository()
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	failing := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "group-1"}
	succeeding := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{failing, succeeding}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{
		{RunID: run.ID, Index: 0, Action: failing, Outcome: igdomain.WorkflowStepPending},
		{RunID: run.ID, Index: 1, Action: succeeding, Outcome: igdomain.WorkflowStepPending},
	}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params}); err != nil {
		t.Fatal(err)
	}
	stored, err := runs.FindRun(ctx, run.TenantID, run.ID)
	if err != nil || stored.Status != igdomain.WorkflowRunPartiallyFailed {
		t.Fatalf("run status = %#v, %v, want partially_failed", stored, err)
	}
}

// wi-222: add_group_member against a User who is already a member must report
// no_op, not changed, so dry-run and the real run agree.
func TestLifecycleWorkflowRunHandlerAddGroupMemberNoOpWhenAlreadyMember(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := idmmemory.NewUserRepository()
	groups := idmmemory.NewGroupRepository()
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
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: group.ID}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: action, Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users, GroupRepo: groups})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params}); err != nil {
		t.Fatal(err)
	}
	storedSteps, err := runs.ListSteps(ctx, run.TenantID, run.ID)
	if err != nil || storedSteps[0].Outcome != igdomain.WorkflowStepNoop {
		t.Fatalf("steps = %#v, %v, want no_op", storedSteps, err)
	}
}

// wi-222: unassign_application against a User who has no assignment must
// report no_op rather than unconditionally claiming changed.
func TestLifecycleWorkflowRunHandlerUnassignApplicationNoOpWhenNotAssigned(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := idmmemory.NewUserRepository()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	user := &idmdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	app := &appdomain.Application{TenantID: "tenant-a", ApplicationID: "app-1", Name: "Payroll", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionUnassignApplication, ApplicationID: app.ApplicationID}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: action, Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users, ApplicationRepo: apps, AssignmentRepo: assignments})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params}); err != nil {
		t.Fatal(err)
	}
	storedSteps, err := runs.ListSteps(ctx, run.TenantID, run.ID)
	if err != nil || storedSteps[0].Outcome != igdomain.WorkflowStepNoop {
		t.Fatalf("steps = %#v, %v, want no_op", storedSteps, err)
	}
}
