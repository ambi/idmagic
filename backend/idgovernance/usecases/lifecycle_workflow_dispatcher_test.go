package usecases_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	igmemory "github.com/ambi/idmagic/backend/idgovernance/db_memory"
	igdomain "github.com/ambi/idmagic/backend/idgovernance/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/ports"
	"github.com/ambi/idmagic/backend/idgovernance/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type failOnceJobRepository struct {
	jobsports.JobRepository
	fail bool
}

//spec:covers REQ-IDGOVERNANCE-003: 利用者の正式な変更経路で捕捉したワークフローをジョブとして実行し、宣言した効果まで到達させる。
func TestUserChangeRunsLifecycleWorkflowToDeclaredEffects(t *testing.T) {
	ctx := workflowContext()
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	workflows := igmemory.NewLifecycleWorkflowRepository()
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	applications := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	jobs := jobsmemory.NewJobRepository()

	group := &groupdomain.Group{ID: "engineering", TenantID: "tenant-a", Name: "Engineering", MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now}
	if err := groups.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	application := &appdomain.Application{TenantID: "tenant-a", ID: "employee-portal", Name: "Employee portal", Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now}
	if err := applications.Save(ctx, application); err != nil {
		t.Fatal(err)
	}
	workflow, err := usecases.CreateLifecycleWorkflow(ctx, usecases.LifecycleWorkflowDeps{Repo: workflows, GroupRepo: groups, ApplicationRepo: applications}, usecases.CreateLifecycleWorkflowInput{
		Name:    "Joiner",
		Trigger: igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		Actions: []igdomain.WorkflowAction{
			{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: group.ID},
			{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: application.ID},
		},
		Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.EnableLifecycleWorkflow(ctx, usecases.LifecycleWorkflowDeps{Repo: workflows, GroupRepo: groups, ApplicationRepo: applications}, workflow.ID, workflow.CurrentRevision, "admin", now); err != nil {
		t.Fatal(err)
	}

	created, err := userusecases.CreateUser(ctx, userusecases.AdminUserDeps{
		UserRepo: users,
		UserMutationCommitter: usecases.UserMutationCommitter{
			WorkflowRepo: workflows,
			RunRepo:      runs,
			UserRepo:     users,
			Capture:      &igmemory.UserWorkflowCapture{Users: users, Runs: runs},
		},
		PasswordHasher:      testing_passwords.NewHasher(),
		PasswordHistoryRepo: passwordmemory.NewPasswordHistoryRepository(),
	}, userusecases.CreateUserInput{ActorUserID: "admin", PreferredUsername: "alice", Password: "initial-password-9182", Now: now.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(ctx, usecases.LifecycleWorkflowDispatcherDeps{RunRepo: runs, JobRepo: jobs}, 10, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	claimed, err := jobs.ClaimBatch(ctx, "worker-1", jobsdomain.LaneDefault, 1, time.Minute, now.Add(3*time.Minute))
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimBatch = %#v, %v", claimed, err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{
		RunRepo: runs, UserRepo: users, GroupRepo: groups, ApplicationRepo: applications, AssignmentRepo: assignments,
		ApplicationAssignments: desiredStateAssignments(applications, assignments, users, groups, nil),
	})
	result, err := handler(ctx, claimed[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs.Complete(ctx, claimed[0].ID, "worker-1", result, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}

	memberships, err := groups.ListGroupsByUser(ctx, "tenant-a", created.ID)
	if err != nil || len(memberships) != 1 || memberships[0].ID != group.ID {
		t.Fatalf("memberships = %#v, %v", memberships, err)
	}
	applicationAssignments, err := assignments.ListBySubjects(ctx, "tenant-a", []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: created.ID}})
	if err != nil || len(applicationAssignments) != 1 || applicationAssignments[0].ApplicationID != application.ID {
		t.Fatalf("assignments = %#v, %v", applicationAssignments, err)
	}
}

func (r *failOnceJobRepository) Enqueue(ctx context.Context, in jobsports.EnqueueInput) (*jobsdomain.Job, bool, error) {
	if r.fail {
		r.fail = false
		return nil, false, errors.New("transient enqueue failure")
	}
	return r.JobRepository.Enqueue(ctx, in)
}

// queuedDisableRun は tenant-a の user-1 を無効化する、job_id 未設定で queued の実行を保存する。
func queuedDisableRun(t *testing.T, runs *igmemory.LifecycleWorkflowRunRepository, users *usermemory.UserRepository, now time.Time) *igdomain.WorkflowRun {
	t.Helper()
	if users != nil {
		user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
		if err := users.Save(context.Background(), user); err != nil {
			t.Fatal(err)
		}
	}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: "user-1", TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{{Kind: igdomain.WorkflowActionDisableUser}}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(context.Background(), run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	return run
}

//spec:covers EX-JOBS-008-01: 即時の投入に失敗した queued の実行を、ディスパッチャーが dedup_key=lifecycle-workflow-run:{run_id} で投入して job_id を関連付け、worker がその Job を取得してハンドラーを実行する。1 回目の投入は API プロセスの即時投入に当たり、失敗させる。2 回目は worker の定期ディスパッチャーの再走査に当たる。
func TestDispatchQueuedLifecycleWorkflowRunsAttachesDeduplicatedJob(t *testing.T) {
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	run := queuedDisableRun(t, runs, users, now)
	jobRepo := jobsmemory.NewJobRepository()
	jobs := &failOnceJobRepository{JobRepository: jobRepo, fail: true}
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
	job, err := jobRepo.Get(context.Background(), *stored.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if job.DedupKey == nil || *job.DedupKey != "lifecycle-workflow-run:run-1" || job.Kind != usecases.LifecycleWorkflowRunJobKind || job.TenantID != "tenant-a" {
		t.Fatalf("dispatched job = kind %q tenant %q dedup %v, want the run's deduplicated lifecycle job", job.Kind, job.TenantID, job.DedupKey)
	}

	handlers := jobsusecases.NewHandlerRegistry()
	handlers.Register(usecases.LifecycleWorkflowRunJobKind, usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{RunRepo: runs, UserRepo: users}))
	runner := jobsusecases.NewRunner(
		jobsusecases.RunnerConfig{WorkerID: "worker-1", Lane: jobsdomain.LaneDefault, PollInterval: 5 * time.Millisecond, LeaseDuration: time.Minute},
		jobsusecases.RunnerDeps{Repo: jobRepo, Handlers: handlers},
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if job, err = jobRepo.Get(context.Background(), job.ID); err != nil || job.Status == jobsdomain.StatusSucceeded {
			break
		}
	}
	cancel()
	<-done
	if err != nil || job.Status != jobsdomain.StatusSucceeded {
		t.Fatalf("dispatched job status = %q, %v; want succeeded", job.Status, err)
	}
	user, err := users.FindBySub(context.Background(), "user-1")
	if err != nil || user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("user after the handler ran = %#v, %v; want disabled", user, err)
	}
}

// staleRunListing は、別のディスパッチャーが job_id を関連付ける前に読んだ一覧を返し続ける。
type staleRunListing struct {
	igports.LifecycleWorkflowRunRepository
	listed []*igdomain.WorkflowRun
}

func (s staleRunListing) ListUnenqueuedRuns(context.Context, int) ([]*igdomain.WorkflowRun, error) {
	return s.listed, nil
}

//spec:covers EX-JOBS-007-01: 同じ実行を重複して投入するディスパッチャーは、同じ dedup_key の既存 Job を受け取り、lifecycle_workflow_run の Job を新しく作らない。
func TestDuplicateDispatchReusesTheLifecycleWorkflowJob(t *testing.T) {
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	run := queuedDisableRun(t, runs, nil, now)
	jobs := jobsmemory.NewJobRepository()
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(context.Background(), usecases.LifecycleWorkflowDispatcherDeps{RunRepo: runs, JobRepo: jobs}, 10, now); err != nil {
		t.Fatal(err)
	}
	first, err := runs.FindRun(context.Background(), "tenant-a", run.ID)
	if err != nil || first.JobID == nil {
		t.Fatalf("first dispatch attachment = %#v, %v", first, err)
	}

	stale := staleRunListing{LifecycleWorkflowRunRepository: runs, listed: []*igdomain.WorkflowRun{run}}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(context.Background(), usecases.LifecycleWorkflowDispatcherDeps{RunRepo: stale, JobRepo: jobs}, 10, now.Add(time.Second)); err != nil {
		t.Fatalf("duplicate dispatch: %v", err)
	}
	queued, err := jobs.ListByTenantAndKinds(context.Background(), "tenant-a", []jobsdomain.JobKind{usecases.LifecycleWorkflowRunJobKind}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(queued) != 1 || queued[0].ID != *first.JobID {
		t.Fatalf("jobs after a duplicate dispatch = %d (first %q), want only the original job", len(queued), *first.JobID)
	}
	after, err := runs.FindRun(context.Background(), "tenant-a", run.ID)
	if err != nil || after.JobID == nil || *after.JobID != *first.JobID {
		t.Fatalf("run job after a duplicate dispatch = %#v, %v; want the original job", after, err)
	}
}

func TestLifecycleWorkflowRunHandlerCheckpointsAndSkipsCompletedStepsOnRetry(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
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
	users := usermemory.NewUserRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
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
	// イベントだけでは再開可能性が担保されない。成功した step の結果が
	// checkpoint として保存されていることを確かめる。
	persisted, err := runs.ListSteps(ctx, run.TenantID, run.ID)
	if err != nil || len(persisted) != 1 {
		t.Fatalf("ListSteps = %+v, err = %v", persisted, err)
	}
	if persisted[0].Outcome == igdomain.WorkflowStepPending {
		t.Fatalf("checkpointed step outcome = %v, want a terminal outcome", persisted[0].Outcome)
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
	users := usermemory.NewUserRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
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
	users := usermemory.NewUserRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
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
//
//spec:covers EX-IDGOVERNANCE-005-01: 既にメンバーである User への add_group_member は no_op になり、メンバーシップの行を重ねず、WorkflowRun は succeeded で終わることを固定する。
func TestLifecycleWorkflowRunHandlerAddGroupMemberNoOpWhenAlreadyMember(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	group := &groupdomain.Group{ID: "group-1", TenantID: "tenant-a", Name: "Engineering", MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now}
	if err := groups.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	if ok, err := groups.AddMember(ctx, &groupdomain.GroupMember{GroupID: group.ID, UserID: user.ID, CreatedAt: now}); err != nil || !ok {
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
	if members, err := groups.ListMembersByGroup(ctx, "tenant-a", group.ID); err != nil || len(members) != 1 {
		t.Fatalf("members = %#v, %v, want the single seeded membership", members, err)
	}
	if stored, err := runs.FindRun(ctx, run.TenantID, run.ID); err != nil || stored.Status != igdomain.WorkflowRunSucceeded {
		t.Fatalf("run = %#v, %v, want succeeded", stored, err)
	}
}

// wi-222: unassign_application against a User who has no assignment must
// report no_op rather than unconditionally claiming changed.
func TestLifecycleWorkflowRunHandlerUnassignApplicationNoOpWhenNotAssigned(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Roles: []string{"member"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	app := &appdomain.Application{TenantID: "tenant-a", ID: "app-1", Name: "Payroll", Kind: appdomain.ApplicationFederated, Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now}
	if err := apps.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionUnassignApplication, ApplicationID: app.ID}
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

// 直接割り当てが指定と異なる visibility で存在するとき、assign_application は事前の評価で
// no_op と判定せず、実行で指定どおりの visibility へ更新する。
func TestLifecycleWorkflowRunHandlerAssignApplicationUpdatesADifferentVisibility(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	user := &userdomain.User{ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	if err := apps.Save(ctx, &appdomain.Application{TenantID: "tenant-a", ID: "app-1", Name: "Payroll", Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	// 別の Application への hidden の直接割り当ては、app-1 の判定に混ざってはならない。
	for _, seeded := range []appdomain.ApplicationAssignment{
		{TenantID: "tenant-a", ApplicationID: "app-1", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: user.ID, Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now},
		{TenantID: "tenant-a", ApplicationID: "app-2", SubjectType: appdomain.AssignmentSubjectUser, SubjectID: user.ID, Visibility: appdomain.AssignmentHidden, CreatedAt: now, UpdatedAt: now},
	} {
		if err := assignments.Save(ctx, &seeded); err != nil {
			t.Fatal(err)
		}
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: "app-1", Visibility: "hidden"}
	evalDeps := usecases.LifecycleActionEvalDeps{ApplicationRepo: apps, AssignmentRepo: assignments}
	if outcome, _, err := usecases.EvaluateLifecycleAction(ctx, evalDeps, "tenant-a", user, action); err != nil || outcome != igdomain.WorkflowActionWouldChange {
		t.Fatalf("EvaluateLifecycleAction() = %s, %v; want would_change", outcome, err)
	}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	if _, err := runs.SaveRun(ctx, run, []igdomain.WorkflowStep{{RunID: run.ID, Action: action, Outcome: igdomain.WorkflowStepPending}}); err != nil {
		t.Fatal(err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{
		RunRepo: runs, UserRepo: users, ApplicationRepo: apps, AssignmentRepo: assignments,
		ApplicationAssignments: desiredStateAssignments(apps, assignments, users, groupmemory.NewGroupRepository(), nil),
	})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
		t.Fatal(err)
	}
	if steps, err := runs.ListSteps(ctx, run.TenantID, run.ID); err != nil || steps[0].Outcome != igdomain.WorkflowStepChanged {
		t.Fatalf("steps = %#v, %v; want changed", steps, err)
	}
	stored, err := assignments.ListBySubjects(ctx, "tenant-a", []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: user.ID}})
	if err != nil {
		t.Fatal(err)
	}
	for _, assignment := range stored {
		if assignment.Visibility != appdomain.AssignmentHidden {
			t.Fatalf("assignment %s visibility = %s, want hidden", assignment.ApplicationID, assignment.Visibility)
		}
	}
}

// scenario `Tenancy: テナントの通知テンプレート上書きは組込み既定より優先される`
// の前提となるカタログ接続。send_email action は template_key を件名と本文に直挿しせず、
// カタログの LifecycleWorkflowNotification から解決する。
func TestLifecycleWorkflowRunHandlerSendsCatalogTemplateForSendEmail(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	runs := igmemory.NewLifecycleWorkflowRunRepository()
	users := usermemory.NewUserRepository()
	email := "alice@example.test"
	locale := "ja"
	user := &userdomain.User{
		ID: "user-1", TenantID: "tenant-a", PreferredUsername: "alice", PasswordHash: "hash",
		Email: &email, EmailVerified: true, Roles: []string{"member"},
		Lifecycle:  userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]userdomain.AttributeValue{"locale": {Type: idmdomain.AttributeTypeString, String: &locale}},
		CreatedAt:  now, UpdatedAt: now,
	}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionSendEmail, TemplateKey: "welcome"}
	run := &igdomain.WorkflowRun{ID: "run-1", TenantID: "tenant-a", WorkflowID: "workflow-1", Revision: 1, SourceOccurrenceID: "source-1", TargetUserID: user.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: []igdomain.WorkflowAction{action}, Status: igdomain.WorkflowRunQueued, TriggeredAt: now}
	steps := []igdomain.WorkflowStep{{RunID: run.ID, Index: 0, Action: action, Outcome: igdomain.WorkflowStepPending}}
	if created, err := runs.SaveRun(ctx, run, steps); err != nil || !created {
		t.Fatalf("SaveRun = %v, %v", created, err)
	}
	sender := &email_memory.NoopEmailSender{}
	handler := usecases.LifecycleWorkflowRunHandler(usecases.LifecycleWorkflowExecutorDeps{
		RunRepo: runs, UserRepo: users,
		Notifier: &template.Notifier{Sender: sender, SystemDefaultLocale: "en"},
	})
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := handler(ctx, &jobsdomain.Job{TenantID: run.TenantID, Params: params}); err != nil {
		t.Fatal(err)
	}

	if len(sender.Sent) != 1 {
		t.Fatalf("sent emails=%d, want 1", len(sender.Sent))
	}
	sent := sender.Sent[0]
	if sent.Subject == "welcome" || sent.Text == "welcome" {
		t.Fatalf("the template key leaked into the message: subject=%q text=%q", sent.Subject, sent.Text)
	}
	if sent.Text == "" || sent.HTML == "" {
		t.Errorf("both parts are required, got text=%q html=%q", sent.Text, sent.HTML)
	}
	if !strings.Contains(sent.Text, "welcome") {
		t.Errorf("the template key should appear as a rendered variable: %q", sent.Text)
	}
	if !strings.Contains(sent.Text, "alice") {
		t.Errorf("text body has no recipient display name: %q", sent.Text)
	}
}
