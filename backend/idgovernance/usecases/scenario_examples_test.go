package usecases_test

// IdGovernance の実行系の具体例を、User の変更は IdManagement のユースケース、実行は jobs の
// Runner またはハンドラーを通して観測する。WorkflowRun とステップの状態は保存先から読み直す。

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
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
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// eventLog は Runner の goroutine から発行されるイベントを集める。
type eventLog struct {
	mu     sync.Mutex
	events []spec.DomainEvent
}

func (l *eventLog) emit(event spec.DomainEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
	return nil
}

func (l *eventLog) count(eventType string) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	n := 0
	for _, event := range l.events {
		if event.EventType() == eventType {
			n++
		}
	}
	return n
}

// hookedUsers は User の保存のあとに差し込みを呼ぶ。ステップの実行中に管理者の操作が
// 割り込む状況と、保存の回数を観測するために使う。
type hookedUsers struct {
	*usermemory.UserRepository
	mu     sync.Mutex
	saves  int
	onSave func()
}

func (u *hookedUsers) Save(ctx context.Context, user *userdomain.User) error {
	if err := u.UserRepository.Save(ctx, user); err != nil {
		return err
	}
	u.mu.Lock()
	u.saves++
	hook := u.onSave
	u.onSave = nil
	u.mu.Unlock()
	if hook != nil {
		hook()
	}
	return nil
}

func (u *hookedUsers) saveCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.saves
}

// timingOutGroups は最初の AddMember だけを保存先のタイムアウトとして失敗させる。
type timingOutGroups struct {
	*groupmemory.GroupRepository
	mu      sync.Mutex
	timeout bool
}

func (g *timingOutGroups) AddMember(ctx context.Context, member *groupdomain.GroupMember) (bool, error) {
	g.mu.Lock()
	timeout := g.timeout
	g.timeout = false
	g.mu.Unlock()
	if timeout {
		return false, context.DeadlineExceeded
	}
	return g.GroupRepository.AddMember(ctx, member)
}

type governanceFixture struct {
	ctx         context.Context
	now         time.Time
	workflows   *igmemory.LifecycleWorkflowRepository
	runs        *igmemory.LifecycleWorkflowRunRepository
	users       *hookedUsers
	groups      *timingOutGroups
	apps        *appmemory.ApplicationRepository
	assignments *appmemory.ApplicationAssignmentRepository
	jobs        *jobsmemory.JobRepository
	sender      *email_memory.NoopEmailSender
	events      *eventLog
}

func newGovernanceFixture(t *testing.T) *governanceFixture {
	t.Helper()
	return &governanceFixture{
		ctx: workflowContext(), now: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC),
		workflows: igmemory.NewLifecycleWorkflowRepository(), runs: igmemory.NewLifecycleWorkflowRunRepository(),
		users:       &hookedUsers{UserRepository: usermemory.NewUserRepository()},
		groups:      &timingOutGroups{GroupRepository: groupmemory.NewGroupRepository()},
		apps:        appmemory.NewApplicationRepository(),
		assignments: appmemory.NewApplicationAssignmentRepository(),
		jobs:        jobsmemory.NewJobRepository(), sender: &email_memory.NoopEmailSender{}, events: &eventLog{},
	}
}

func (f *governanceFixture) workflowDeps() usecases.LifecycleWorkflowDeps {
	return usecases.LifecycleWorkflowDeps{Repo: f.workflows, RunRepo: f.runs, GroupRepo: f.groups, ApplicationRepo: f.apps, Emit: f.events.emit}
}

func (f *governanceFixture) userDeps() userusecases.AdminUserDeps {
	return userusecases.AdminUserDeps{
		UserRepo: f.users,
		UserMutationCommitter: usecases.UserMutationCommitter{
			WorkflowRepo: f.workflows, RunRepo: f.runs, UserRepo: f.users,
			Capture: &igmemory.UserWorkflowCapture{Users: f.users.UserRepository, Runs: f.runs},
		},
		PasswordHasher: testing_passwords.NewHasher(), PasswordHistoryRepo: passwordmemory.NewPasswordHistoryRepository(),
	}
}

// executor は worker が組み立てるのと同じ依存を渡す。
func (f *governanceFixture) executor() usecases.LifecycleWorkflowExecutorDeps {
	return usecases.LifecycleWorkflowExecutorDeps{
		RunRepo: f.runs, WorkflowRepo: f.workflows, UserRepo: f.users, GroupRepo: f.groups,
		ApplicationRepo: f.apps, AssignmentRepo: f.assignments,
		ApplicationAssignments: desiredStateAssignments(f.apps, f.assignments, f.users, f.groups, f.events.emit),
		Notifier:               &template.Notifier{Sender: f.sender, SystemDefaultLocale: "en"},
		Emit:                   f.events.emit,
	}
}

// desiredStateAssignments は worker と同じく Application の Module から割り当て操作を組み立てる。
func desiredStateAssignments(apps appports.ApplicationRepository, assignments appports.AssignmentRepository, users userports.UserRepository, groups groupports.GroupRepository, emit func(spec.DomainEvent) error) igports.ApplicationAssignments {
	return application.Module{Repo: apps, AssignmentRepo: assignments}.DesiredStateAssignments(users, groups, func(event spec.DomainEvent) {
		if emit != nil {
			_ = emit(event)
		}
	}, "lifecycle-workflow")
}

func (f *governanceFixture) seedGroup(t *testing.T, tenantID, id string) {
	t.Helper()
	if err := f.groups.Save(f.ctx, &groupdomain.Group{ID: id, TenantID: tenantID, Name: id, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: f.now, UpdatedAt: f.now}); err != nil {
		t.Fatal(err)
	}
}

func (f *governanceFixture) seedApplication(t *testing.T, id string) {
	t.Helper()
	if err := f.apps.Save(f.ctx, &appdomain.Application{TenantID: "tenant-a", ID: id, Name: id, Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive, CreatedAt: f.now, UpdatedAt: f.now}); err != nil {
		t.Fatal(err)
	}
}

func (f *governanceFixture) enabledWorkflow(t *testing.T, trigger igdomain.WorkflowTrigger, actions ...igdomain.WorkflowAction) *igdomain.LifecycleWorkflow {
	t.Helper()
	workflow, err := usecases.CreateLifecycleWorkflow(f.ctx, f.workflowDeps(), usecases.CreateLifecycleWorkflowInput{Name: "workflow-" + string(trigger.Kind), Trigger: trigger, Actions: actions, Now: f.now})
	if err != nil {
		t.Fatal(err)
	}
	if workflow, err = usecases.EnableLifecycleWorkflow(f.ctx, f.workflowDeps(), workflow.ID, workflow.CurrentRevision, "admin", f.now); err != nil {
		t.Fatal(err)
	}
	return workflow
}

func (f *governanceFixture) createUser(t *testing.T, username string) *userdomain.User {
	t.Helper()
	user, err := userusecases.CreateUser(f.ctx, f.userDeps(), userusecases.CreateUserInput{ActorUserID: "admin", PreferredUsername: username, Password: "initial-password-9182", Now: f.now})
	if err != nil {
		t.Fatal(err)
	}
	return user
}

func (f *governanceFixture) setDepartment(t *testing.T, userID, department string) {
	t.Helper()
	attributes := map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}
	if _, err := userusecases.UpdateUser(f.ctx, f.userDeps(), userusecases.UpdateUserInput{Sub: userID, Attributes: &attributes, Now: f.now}); err != nil {
		t.Fatal(err)
	}
}

func (f *governanceFixture) runsOf(t *testing.T, workflowID string) []*igdomain.WorkflowRun {
	t.Helper()
	runs, err := f.runs.ListRuns(f.ctx, "tenant-a", workflowID, 0)
	if err != nil {
		t.Fatal(err)
	}
	return runs
}

func (f *governanceFixture) outcomes(t *testing.T, runID string) []igdomain.WorkflowStepOutcome {
	t.Helper()
	steps, err := f.runs.ListSteps(f.ctx, "tenant-a", runID)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]igdomain.WorkflowStepOutcome, len(steps))
	for i, step := range steps {
		out[i] = step.Outcome
	}
	return out
}

// runWorker は dispatcher で Job を投入し、worker と同じ Runner に、その Job が終わるまで
// 実行させる。バックオフは再試行を待たずに観測できるよう短くする。
func (f *governanceFixture) runWorker(t *testing.T, run *igdomain.WorkflowRun) *jobsdomain.Job {
	t.Helper()
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(f.ctx, usecases.LifecycleWorkflowDispatcherDeps{RunRepo: f.runs, JobRepo: f.jobs}, 10, f.now); err != nil {
		t.Fatal(err)
	}
	stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID)
	if err != nil || stored.JobID == nil {
		t.Fatalf("run job attachment = %+v, %v", stored, err)
	}
	handlers := jobsusecases.NewHandlerRegistry()
	handlers.Register(usecases.LifecycleWorkflowRunJobKind, usecases.LifecycleWorkflowRunHandler(f.executor()))
	runner := jobsusecases.NewRunner(
		jobsusecases.RunnerConfig{WorkerID: "worker-1", Lane: jobsdomain.LaneDefault, PollInterval: 2 * time.Millisecond, LeaseDuration: time.Minute, BackoffBase: time.Millisecond, BackoffCap: time.Millisecond},
		jobsusecases.RunnerDeps{Repo: f.jobs, Handlers: handlers},
	)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	var job *jobsdomain.Job
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Millisecond) {
		if job, err = f.jobs.Get(context.Background(), *stored.JobID); err != nil || jobFinished(job) {
			break
		}
	}
	cancel()
	<-done
	if err != nil || !jobFinished(job) {
		t.Fatalf("job = %+v, %v; want it to finish", job, err)
	}
	return job
}

func onlyRun(t *testing.T, runs []*igdomain.WorkflowRun) *igdomain.WorkflowRun {
	t.Helper()
	if len(runs) != 1 {
		t.Fatalf("runs = %d, want exactly one", len(runs))
	}
	return runs[0]
}

//spec:covers EX-IDGOVERNANCE-003-01: department の変更が changed_fields=["department"] を保持する WorkflowRun を作り、worker の実行で add_group_member と assign_application が changed、WorkflowRun が succeeded になり、LifecycleWorkflowRunSucceeded が発行され、メンバーシップと割り当てが実際に作られることを固定する。
func TestDepartmentChangeRunsTheWorkflowToItsDeclaredEffects(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "engineering")
	f.seedApplication(t, "engineering-portal")
	alice := f.createUser(t, "alice")
	f.setDepartment(t, alice.ID, "Sales")
	workflow := f.enabledWorkflow(t,
		igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserAttributesChanged, WatchedAttributes: []string{"department"}},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "engineering"},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: "engineering-portal"},
	)

	f.setDepartment(t, alice.ID, "Engineering")
	run := onlyRun(t, f.runsOf(t, workflow.ID))
	if !slices.Equal(run.ChangedFields, []string{"department"}) {
		t.Fatalf("changed_fields = %v, want [department]", run.ChangedFields)
	}
	f.runWorker(t, run)

	if got := f.outcomes(t, run.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepChanged, igdomain.WorkflowStepChanged}) {
		t.Fatalf("step outcomes = %v, want [changed changed]", got)
	}
	if stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID); err != nil || stored.Status != igdomain.WorkflowRunSucceeded {
		t.Fatalf("run = %+v, %v; want succeeded", stored, err)
	}
	if n := f.events.count("LifecycleWorkflowRunSucceeded"); n != 1 {
		t.Fatalf("LifecycleWorkflowRunSucceeded emitted %d times, want 1", n)
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-a", "engineering"); err != nil || len(members) != 1 || members[0].UserID != alice.ID {
		t.Fatalf("members = %+v, %v", members, err)
	}
	if assigned, err := f.assignments.ListBySubjects(f.ctx, "tenant-a", []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: alice.ID}}); err != nil || len(assigned) != 1 {
		t.Fatalf("assignments = %+v, %v", assigned, err)
	}
}

//spec:covers EX-IDGOVERNANCE-004-01: 一致する User の変更が、重複排除キーの各要素を持つ queued の WorkflowRun と pending のステップを捕捉し、同じ発火事象の再配信は既存の WorkflowRun に収束してステップを増やさず、dispatcher が投入した lifecycle_workflow_run の Job が WorkflowRun に関連付けられることを固定する。同一トランザクションであることは db_postgres のテストが観測する。
func TestUserMutationCapturesARunThatConvergesOnRedeliveryAndGetsAJob(t *testing.T) {
	f := newGovernanceFixture(t)
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionSendEmail, TemplateKey: "welcome"},
	)
	alice := f.createUser(t, "alice")

	run := onlyRun(t, f.runsOf(t, workflow.ID))
	if run.Status != igdomain.WorkflowRunQueued || run.JobID != nil || run.TenantID != "tenant-a" || run.WorkflowID != workflow.ID || run.Revision != 1 || run.SourceOccurrenceID == "" || run.TargetUserID != alice.ID {
		t.Fatalf("captured run = %+v, want a queued run carrying the whole dedup key", run)
	}
	if got := f.outcomes(t, run.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepPending, igdomain.WorkflowStepPending}) {
		t.Fatalf("steps = %v, want two pending steps", got)
	}

	redelivered := *run
	redelivered.ID = "redelivered-run"
	created, err := f.runs.SaveRun(f.ctx, &redelivered, []igdomain.WorkflowStep{
		{RunID: redelivered.ID, Index: 0, Action: run.Actions[0], Outcome: igdomain.WorkflowStepPending},
		{RunID: redelivered.ID, Index: 1, Action: run.Actions[1], Outcome: igdomain.WorkflowStepPending},
	})
	if err != nil || created {
		t.Fatalf("redelivery SaveRun = %v, %v; want it to converge on the existing run", created, err)
	}
	onlyRun(t, f.runsOf(t, workflow.ID))
	if steps, err := f.runs.ListSteps(f.ctx, "tenant-a", redelivered.ID); err != nil || len(steps) != 0 {
		t.Fatalf("redelivered steps = %d, %v; want none", len(steps), err)
	}

	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(f.ctx, usecases.LifecycleWorkflowDispatcherDeps{RunRepo: f.runs, JobRepo: f.jobs}, 10, f.now); err != nil {
		t.Fatal(err)
	}
	stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID)
	if err != nil || stored.JobID == nil {
		t.Fatalf("run after dispatch = %+v, %v", stored, err)
	}
	if job, err := f.jobs.Get(f.ctx, *stored.JobID); err != nil || job.Kind != usecases.LifecycleWorkflowRunJobKind {
		t.Fatalf("attached job = %+v, %v", job, err)
	}
}

//spec:covers EX-IDGOVERNANCE-004-02: 捕捉した WorkflowRun の投入が一時的に失敗すると job_id 未設定の queued のまま残り、定期ディスパッチャーの再走査が Job を関連付け、重ねて走査しても Job が 1 件のままであることを固定する。
func TestTransientEnqueueFailureLeavesTheRunForTheDispatcher(t *testing.T) {
	f := newGovernanceFixture(t)
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated}, igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser})
	f.createUser(t, "alice")
	run := onlyRun(t, f.runsOf(t, workflow.ID))

	failing := &failOnceJobRepository{JobRepository: f.jobs, fail: true}
	if err := usecases.DispatchQueuedLifecycleWorkflowRuns(f.ctx, usecases.LifecycleWorkflowDispatcherDeps{RunRepo: f.runs, JobRepo: failing}, 10, f.now); err == nil {
		t.Fatal("the failed enqueue must surface")
	}
	if stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID); err != nil || stored.Status != igdomain.WorkflowRunQueued || stored.JobID != nil {
		t.Fatalf("run after the failed enqueue = %+v, %v; want queued without job_id", stored, err)
	}
	for range 2 {
		if err := usecases.DispatchQueuedLifecycleWorkflowRuns(f.ctx, usecases.LifecycleWorkflowDispatcherDeps{RunRepo: f.runs, JobRepo: failing}, 10, f.now); err != nil {
			t.Fatal(err)
		}
	}
	stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID)
	if err != nil || stored.JobID == nil {
		t.Fatalf("run after the rescan = %+v, %v", stored, err)
	}
	jobs, err := f.jobs.ListByTenantAndKinds(f.ctx, "tenant-a", []jobsdomain.JobKind{usecases.LifecycleWorkflowRunJobKind}, 0)
	if err != nil || len(jobs) != 1 || jobs[0].ID != *stored.JobID {
		t.Fatalf("jobs = %d, %v; want only the attached job", len(jobs), err)
	}
}

//spec:covers EX-IDGOVERNANCE-006-01: department に同じ値を指定した更新は WorkflowRun を作らず、異なる値への更新なら作ることを対照として固定する。動的グループの拒否は handlers_http のテストが観測する。
func TestUnchangedAttributeValueDoesNotTriggerTheWorkflow(t *testing.T) {
	f := newGovernanceFixture(t)
	alice := f.createUser(t, "alice")
	f.setDepartment(t, alice.ID, "Engineering")
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserAttributesChanged, WatchedAttributes: []string{"department"}}, igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser})

	f.setDepartment(t, alice.ID, "Engineering")
	if runs := f.runsOf(t, workflow.ID); len(runs) != 0 {
		t.Fatalf("runs = %d, want none for an unchanged value", len(runs))
	}
	f.setDepartment(t, alice.ID, "Sales")
	onlyRun(t, f.runsOf(t, workflow.ID))
}

//spec:covers EX-IDGOVERNANCE-008-01: 退職相当のステータス変更で作られた WorkflowRun は、検証済みメールアドレスが無くても disable_user と remove_group_member を changed にして実際にアクセスを剥奪し、send_email はブロックされた失敗、WorkflowRun は partially_failed になり、LifecycleWorkflowRunPartiallyFailed と LifecycleWorkflowStepFailed が発行されることを固定する。
func TestLeaverWorkflowRevokesAccessEvenWhenTheNotificationIsBlocked(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "payroll")
	alice := f.createUser(t, "alice")
	if ok, err := f.groups.AddMember(f.ctx, &groupdomain.GroupMember{GroupID: "payroll", UserID: alice.ID, CreatedAt: f.now}); err != nil || !ok {
		t.Fatalf("seed membership = %v, %v", ok, err)
	}
	active, leaving := idmdomain.UserStatusActive, idmdomain.UserStatusPendingDeletion
	workflow := f.enabledWorkflow(t,
		igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserStatusChanged, FromStatus: &active, ToStatus: &leaving},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionRemoveGroupMember, GroupID: "payroll"},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionSendEmail, TemplateKey: "farewell"},
	)

	if err := userusecases.SoftDeleteUser(f.ctx, f.userDeps(), userusecases.SoftDeleteUserInput{ActorUserID: "admin", Sub: alice.ID, Now: f.now}); err != nil {
		t.Fatal(err)
	}
	run := onlyRun(t, f.runsOf(t, workflow.ID))
	f.runWorker(t, run)

	if got := f.outcomes(t, run.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepChanged, igdomain.WorkflowStepChanged, igdomain.WorkflowStepFailed}) {
		t.Fatalf("step outcomes = %v, want [changed changed failed]", got)
	}
	steps, err := f.runs.ListSteps(f.ctx, "tenant-a", run.ID)
	if err != nil || steps[2].ErrorCode != "notification_unavailable" {
		t.Fatalf("send_email step = %+v, %v; want the blocked notification_unavailable failure", steps, err)
	}
	if stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID); err != nil || stored.Status != igdomain.WorkflowRunPartiallyFailed {
		t.Fatalf("run = %+v, %v; want partially_failed", stored, err)
	}
	if f.events.count("LifecycleWorkflowRunPartiallyFailed") != 1 || f.events.count("LifecycleWorkflowStepFailed") == 0 {
		t.Fatalf("events: partially_failed=%d step_failed=%d", f.events.count("LifecycleWorkflowRunPartiallyFailed"), f.events.count("LifecycleWorkflowStepFailed"))
	}
	if user, err := f.users.FindBySub(f.ctx, alice.ID); err != nil || user.Lifecycle.Status != idmdomain.UserStatusDisabled {
		t.Fatalf("user = %+v, %v; want disabled", user, err)
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-a", "payroll"); err != nil || len(members) != 0 {
		t.Fatalf("members = %+v, %v; want the membership removed", members, err)
	}
	if len(f.sender.Sent) != 0 {
		t.Fatalf("sent = %d, want no email", len(f.sender.Sent))
	}
}

// 試行が残っている間は、失敗したステップがあっても WorkflowRun を終端にせず Jobs へエラーを返す。
func TestFailedStepKeepsTheRunRunningWhileAttemptsRemain(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "engineering")
	f.groups.timeout = true
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated}, igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "engineering"})
	f.createUser(t, "alice")
	run := onlyRun(t, f.runsOf(t, workflow.ID))
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(f.executor())
	if _, err := handler(f.ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 3}); err == nil {
		t.Fatal("a failed step on a non-final attempt must be returned to Jobs for retry")
	}
	if stored, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID); err != nil || stored.Status != igdomain.WorkflowRunRunning {
		t.Fatalf("run = %+v, %v; want it still running", stored, err)
	}
	if f.events.count("LifecycleWorkflowRunFailed") != 0 {
		t.Fatal("the run must not report a terminal failure while attempts remain")
	}
}

//spec:covers EX-IDGOVERNANCE-009-01: 1 回目の試行で保存先がタイムアウトすると Jobs がバックオフ後に同じ Job を再試行し、再試行は changed のステップを再実行せず failed のステップだけを実行して、WorkflowRun の詳細が succeeded になることを固定する。
func TestTransientTimeoutIsRetriedWithoutRepeatingCompletedSteps(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "engineering")
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "engineering"},
	)
	alice := f.createUser(t, "alice")
	run := onlyRun(t, f.runsOf(t, workflow.ID))
	f.groups.timeout = true
	savesBefore := f.users.saveCount()

	job := f.runWorker(t, run)

	if job.Status != jobsdomain.StatusSucceeded || job.Attempts != 2 {
		t.Fatalf("job status=%s attempts=%d, want succeeded on the second attempt", job.Status, job.Attempts)
	}
	if saves := f.users.saveCount() - savesBefore; saves != 1 {
		t.Fatalf("disable_user saved the user %d times, want once (the retry must skip the changed step)", saves)
	}
	view, err := usecases.GetLifecycleWorkflowRun(f.ctx, f.workflowDeps(), run.ID)
	if err != nil || view.Run.Status != igdomain.WorkflowRunSucceeded {
		t.Fatalf("run detail = %+v, %v; want succeeded", view, err)
	}
	if got := f.outcomes(t, run.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepChanged, igdomain.WorkflowStepChanged}) {
		t.Fatalf("step outcomes = %v", got)
	}
	if f.events.count("LifecycleWorkflowRunStarted") != 1 || f.events.count("LifecycleWorkflowStepFailed") != 1 || f.events.count("LifecycleWorkflowRunSucceeded") != 1 {
		t.Fatalf("events started=%d step_failed=%d succeeded=%d", f.events.count("LifecycleWorkflowRunStarted"), f.events.count("LifecycleWorkflowStepFailed"), f.events.count("LifecycleWorkflowRunSucceeded"))
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-a", "engineering"); err != nil || len(members) != 1 || members[0].UserID != alice.ID {
		t.Fatalf("members = %+v, %v", members, err)
	}
}

//spec:covers EX-IDGOVERNANCE-010-01: 同じ対象ユーザーの後続の WorkflowRun は先行が終端になるまでアクションを始めず、各アクションの直前にリソースを同一テナントで再取得し、削除済みのグループと別テナントにしか無いグループを参照するステップは、識別子を含まないエラーコードと failed をチェックポイントに記録し、別テナントのグループには触れないことを固定する。
func TestRunsOfOneUserAreSerializedAndRevalidatedBeforeEachAction(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "first")
	f.seedGroup(t, "tenant-a", "doomed")
	f.seedGroup(t, "tenant-b", "foreign")
	alice := f.createUser(t, "alice")
	queue := func(id string, at time.Time, groups ...string) *igdomain.WorkflowRun {
		actions := []igdomain.WorkflowAction{}
		steps := []igdomain.WorkflowStep{}
		for i, group := range groups {
			action := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: group}
			actions = append(actions, action)
			steps = append(steps, igdomain.WorkflowStep{RunID: id, Index: i, Action: action, Outcome: igdomain.WorkflowStepPending})
		}
		run := &igdomain.WorkflowRun{ID: id, TenantID: "tenant-a", WorkflowID: "workflow-" + id, Revision: 1, SourceOccurrenceID: "occurrence-" + id, TargetUserID: alice.ID, TriggerKind: igdomain.WorkflowTriggerUserCreated, Actions: actions, Status: igdomain.WorkflowRunQueued, TriggeredAt: at}
		if created, err := f.runs.SaveRun(f.ctx, run, steps); err != nil || !created {
			t.Fatalf("SaveRun = %v, %v", created, err)
		}
		return run
	}
	earlier := queue("earlier", f.now, "first")
	later := queue("later", f.now.Add(time.Second), "doomed", "foreign")
	// WorkflowRepo を渡さないのは、この 2 件がワークフローの定義を持たない保存済みの実行だからである。
	executor := f.executor()
	executor.WorkflowRepo = nil
	handler := usecases.LifecycleWorkflowRunHandler(executor)
	job := func(run *igdomain.WorkflowRun) *jobsdomain.Job {
		params, err := json.Marshal(map[string]string{"run_id": run.ID})
		if err != nil {
			t.Fatal(err)
		}
		return &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}
	}

	if _, err := handler(f.ctx, job(later)); err == nil {
		t.Fatal("the later run must wait for the earlier run of the same user")
	}
	if got := f.outcomes(t, later.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepPending, igdomain.WorkflowStepPending}) {
		t.Fatalf("later steps before the earlier run ended = %v, want untouched", got)
	}
	if _, err := handler(f.ctx, job(earlier)); err != nil {
		t.Fatal(err)
	}
	if err := f.groups.Delete(f.ctx, "tenant-a", "doomed"); err != nil {
		t.Fatal(err)
	}
	if _, err := handler(f.ctx, job(later)); err != nil {
		t.Fatal(err)
	}

	steps, err := f.runs.ListSteps(f.ctx, "tenant-a", later.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range steps {
		if step.Outcome != igdomain.WorkflowStepFailed || step.ErrorCode != "resource_not_found" || strings.Contains(step.ErrorCode, step.Action.GroupID) {
			t.Fatalf("step %d = %+v, want failed with a code that names no identifier", step.Index, step)
		}
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-b", "foreign"); err != nil || len(members) != 0 {
		t.Fatalf("foreign members = %+v, %v; want the other tenant untouched", members, err)
	}
}

//spec:covers EX-IDGOVERNANCE-011-01: 無効化で queued の WorkflowRun は直ちに canceled になり、running の WorkflowRun は実行中のステップをチェックポイントした後、次のステップの前に canceled になって次のアクションの効果を残さず、無効化の後の User の変更は WorkflowRun を作らないことを固定する。
func TestDisablingStopsARunningRunAtTheNextStepBoundary(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "engineering")
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "engineering"},
	)
	alice := f.createUser(t, "alice")
	bob := f.createUser(t, "bob")
	var running, queued *igdomain.WorkflowRun
	for _, run := range f.runsOf(t, workflow.ID) {
		switch run.TargetUserID {
		case alice.ID:
			running = run
		case bob.ID:
			queued = run
		}
	}
	// disable_user が alice を保存した直後、つまり 1 つ目のステップの実行中に管理者が無効化する。
	f.users.onSave = func() {
		if _, err := usecases.DisableLifecycleWorkflow(f.ctx, f.workflowDeps(), workflow.ID, workflow.CurrentRevision, "admin", f.now); err != nil {
			t.Errorf("disable: %v", err)
		}
	}
	params, err := json.Marshal(map[string]string{"run_id": running.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.LifecycleWorkflowRunHandler(f.executor())(f.ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 5}); err != nil {
		t.Fatal(err)
	}

	if stored, err := f.runs.FindRun(f.ctx, "tenant-a", queued.ID); err != nil || stored.Status != igdomain.WorkflowRunCanceled {
		t.Fatalf("queued run = %+v, %v; want canceled", stored, err)
	}
	if got := f.outcomes(t, running.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepChanged, igdomain.WorkflowStepCanceled}) {
		t.Fatalf("running run steps = %v, want [changed canceled]", got)
	}
	if stored, err := f.runs.FindRun(f.ctx, "tenant-a", running.ID); err != nil || stored.Status != igdomain.WorkflowRunCanceled {
		t.Fatalf("running run = %+v, %v; want canceled", stored, err)
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-a", "engineering"); err != nil || len(members) != 0 {
		t.Fatalf("members = %+v, %v; want the step after the boundary never to run", members, err)
	}
	if n := f.events.count("LifecycleWorkflowRunCanceled"); n != 2 {
		t.Fatalf("LifecycleWorkflowRunCanceled emitted %d times, want 2", n)
	}
	f.createUser(t, "carol")
	if runs := f.runsOf(t, workflow.ID); len(runs) != 2 {
		t.Fatalf("runs = %d, want no run for a trigger after the disable", len(runs))
	}
}

//spec:covers EX-IDGOVERNANCE-011-02: 再試行を受け付けた後に無効化が競合しても、worker は無効化済みワークフローの WorkflowRun で新しいステップを始めず、WorkflowRun を canceled で終えることを固定する。無効化の後の再試行要求が InvalidRequestError になることは handlers_http のテストが観測する。
func TestARetriedRunOfADisabledWorkflowStartsNoStep(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "engineering")
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated}, igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "engineering"})
	f.createUser(t, "alice")
	run := onlyRun(t, f.runsOf(t, workflow.ID))
	f.groups.timeout = true
	params, err := json.Marshal(map[string]string{"run_id": run.ID})
	if err != nil {
		t.Fatal(err)
	}
	handler := usecases.LifecycleWorkflowRunHandler(f.executor())
	if _, err := handler(f.ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := usecases.RetryLifecycleWorkflowRun(f.ctx, f.workflowDeps(), run.ID); err != nil {
		t.Fatalf("retry while enabled: %v", err)
	}
	// 再試行の Job が取得された後に無効化が届いた状態を作る。queued の取り消しは既に過ぎている。
	stored, err := f.workflows.Find(f.ctx, "tenant-a", workflow.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := stored.Disable(f.now); err != nil {
		t.Fatal(err)
	}
	if err := f.workflows.Save(f.ctx, stored); err != nil {
		t.Fatal(err)
	}

	if _, err := handler(f.ctx, &jobsdomain.Job{TenantID: "tenant-a", Params: params, Attempts: 1, MaxAttempts: 5}); err != nil {
		t.Fatal(err)
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-a", "engineering"); err != nil || len(members) != 0 {
		t.Fatalf("members = %+v, %v; want no new step", members, err)
	}
	if got := f.outcomes(t, run.ID); !slices.Equal(got, []igdomain.WorkflowStepOutcome{igdomain.WorkflowStepCanceled}) {
		t.Fatalf("steps = %v, want [canceled]", got)
	}
	if after, err := f.runs.FindRun(f.ctx, "tenant-a", run.ID); err != nil || after.Status != igdomain.WorkflowRunCanceled {
		t.Fatalf("run = %+v, %v; want canceled", after, err)
	}
	if _, err := usecases.RetryLifecycleWorkflowRun(f.ctx, f.workflowDeps(), run.ID); err == nil {
		t.Fatal("retrying a run of a disabled workflow must be refused")
	}
}

//spec:covers EX-IDGOVERNANCE-013-01: プレビューは対象 User のメンバーシップ、割り当て、必須操作、メール検証状態を実際に読んで no_op、would_change、blocked と理由を返し、WorkflowRun、Job、メンバーシップ、割り当て、必須操作、ステータスを一切変えないことを固定する。
func TestDryRunReadsTheCurrentStateAndChangesNothing(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "engineering")
	f.seedGroup(t, "tenant-a", "doomed")
	f.seedApplication(t, "portal")
	alice := f.createUser(t, "alice")
	if ok, err := f.groups.AddMember(f.ctx, &groupdomain.GroupMember{GroupID: "engineering", UserID: alice.ID, CreatedAt: f.now}); err != nil || !ok {
		t.Fatalf("seed membership = %v, %v", ok, err)
	}
	workflow := f.enabledWorkflow(t, igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "engineering"},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: "portal"},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionSetRequiredAction, RequiredAction: idmdomain.RequiredActionUpdatePassword},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "doomed"},
		igdomain.WorkflowAction{Kind: igdomain.WorkflowActionSendEmail, TemplateKey: "welcome"},
	)
	if err := f.groups.Delete(f.ctx, "tenant-a", "doomed"); err != nil {
		t.Fatal(err)
	}
	runsBefore := len(f.runsOf(t, workflow.ID))
	before, err := f.users.FindBySub(f.ctx, alice.ID)
	if err != nil {
		t.Fatal(err)
	}

	result, err := usecases.DryRunLifecycleWorkflow(f.ctx, usecases.DryRunLifecycleWorkflowDeps{
		Repo: f.workflows, UserRepo: f.users, GroupRepo: f.groups, ApplicationRepo: f.apps, AssignmentRepo: f.assignments,
		Notifier: &template.Notifier{Sender: f.sender, SystemDefaultLocale: "en"},
	}, workflow.ID, alice.ID, f.now)
	if err != nil {
		t.Fatal(err)
	}
	want := []usecases.LifecycleWorkflowDryRunStepResult{
		{ActionKind: igdomain.WorkflowActionAddGroupMember, Outcome: igdomain.WorkflowActionNoOp},
		{ActionKind: igdomain.WorkflowActionAssignApplication, Outcome: igdomain.WorkflowActionWouldChange},
		{ActionKind: igdomain.WorkflowActionSetRequiredAction, Outcome: igdomain.WorkflowActionWouldChange},
		{ActionKind: igdomain.WorkflowActionAddGroupMember, Outcome: igdomain.WorkflowActionBlocked, Reason: "resource_not_found"},
		{ActionKind: igdomain.WorkflowActionSendEmail, Outcome: igdomain.WorkflowActionBlocked, Reason: "notification_unavailable"},
	}
	if !slices.Equal(result.Steps, want) {
		t.Fatalf("steps = %+v, want %+v", result.Steps, want)
	}

	if runs := f.runsOf(t, workflow.ID); len(runs) != runsBefore {
		t.Fatalf("runs = %d, want %d", len(runs), runsBefore)
	}
	if jobs, err := f.jobs.ListByTenantAndKinds(f.ctx, "tenant-a", []jobsdomain.JobKind{usecases.LifecycleWorkflowRunJobKind}, 0); err != nil || len(jobs) != 0 {
		t.Fatalf("jobs = %d, %v; want none", len(jobs), err)
	}
	if members, err := f.groups.ListMembersByGroup(f.ctx, "tenant-a", "engineering"); err != nil || len(members) != 1 {
		t.Fatalf("members = %+v, %v; want only the seeded membership", members, err)
	}
	if assigned, err := f.assignments.ListBySubjects(f.ctx, "tenant-a", []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: alice.ID}}); err != nil || len(assigned) != 0 {
		t.Fatalf("assignments = %+v, %v; want none", assigned, err)
	}
	after, err := f.users.FindBySub(f.ctx, alice.ID)
	if err != nil || !after.UpdatedAt.Equal(before.UpdatedAt) || len(after.Lifecycle.RequiredActions) != 0 || after.Lifecycle.Status != before.Lifecycle.Status || after.EmailVerified != before.EmailVerified {
		t.Fatalf("user after dry-run = %+v, %v; want it unchanged", after, err)
	}
	if len(f.sender.Sent) != 0 {
		t.Fatalf("sent = %d, want no email", len(f.sender.Sent))
	}
}

func jobFinished(job *jobsdomain.Job) bool {
	return job.Status == jobsdomain.StatusSucceeded || job.Status == jobsdomain.StatusFailed || job.Status == jobsdomain.StatusCanceled
}

// 保存と有効化は、フィルターのフィールドとアクションの参照先をワークフローと同じテナントで照合する。
func TestSavingValidatesReferencesInTheWorkflowsTenant(t *testing.T) {
	f := newGovernanceFixture(t)
	f.seedGroup(t, "tenant-a", "manual")
	f.seedGroup(t, "tenant-b", "foreign")
	if err := f.groups.Save(f.ctx, &groupdomain.Group{ID: "dynamic", TenantID: "tenant-a", Name: "dynamic", MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: f.now, UpdatedAt: f.now}); err != nil {
		t.Fatal(err)
	}
	f.seedApplication(t, "portal")
	schemas := usermemory.NewTenantUserAttributeSchemaRepository()
	if err := schemas.Save(f.ctx, &userdomain.TenantUserAttributeSchema{TenantID: "tenant-a", Attributes: []userdomain.UserAttributeDef{{Key: "badge_color", Type: idmdomain.AttributeTypeString}}, CreatedAt: f.now, UpdatedAt: f.now}); err != nil {
		t.Fatal(err)
	}
	deps := f.workflowDeps()
	deps.AttrSchemaRepo = schemas
	filter := func(field string) igdomain.WorkflowTrigger {
		return igdomain.WorkflowTrigger{Kind: igdomain.WorkflowTriggerUserCreated, Filters: []igdomain.WorkflowFilter{{Field: field, Operator: igdomain.WorkflowFilterExists}}}
	}
	disable := igdomain.WorkflowAction{Kind: igdomain.WorkflowActionDisableUser}
	for _, tc := range []struct {
		name    string
		trigger igdomain.WorkflowTrigger
		action  igdomain.WorkflowAction
		valid   bool
	}{
		{"core field", filter("email"), disable, true},
		{"builtin attribute", filter("department"), disable, true},
		{"tenant schema attribute", filter("badge_color"), disable, true},
		{"unknown field", filter("favorite_color"), disable, false},
		{"manual group", filter("email"), igdomain.WorkflowAction{Kind: igdomain.WorkflowActionRemoveGroupMember, GroupID: "manual"}, true},
		{"dynamic group", filter("email"), igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "dynamic"}, false},
		{"other tenant's group", filter("email"), igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "foreign"}, false},
		{"application", filter("email"), igdomain.WorkflowAction{Kind: igdomain.WorkflowActionUnassignApplication, ApplicationID: "portal"}, true},
		{"missing application", filter("email"), igdomain.WorkflowAction{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: "missing"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := usecases.CreateLifecycleWorkflow(f.ctx, deps, usecases.CreateLifecycleWorkflowInput{Name: tc.name, Trigger: tc.trigger, Actions: []igdomain.WorkflowAction{tc.action}, Now: f.now})
			if tc.valid && err != nil {
				t.Fatalf("create = %v, want it accepted", err)
			}
			if !tc.valid && !errors.Is(err, usecases.ErrLifecycleWorkflowInvalidReference) {
				t.Fatalf("create = %v, want ErrLifecycleWorkflowInvalidReference", err)
			}
		})
	}
	unwired := f.workflowDeps()
	unwired.GroupRepo, unwired.ApplicationRepo = nil, nil
	for _, action := range []igdomain.WorkflowAction{
		{Kind: igdomain.WorkflowActionAddGroupMember, GroupID: "manual"},
		{Kind: igdomain.WorkflowActionAssignApplication, ApplicationID: "portal"},
	} {
		if _, err := usecases.CreateLifecycleWorkflow(f.ctx, unwired, usecases.CreateLifecycleWorkflowInput{Name: "unwired-" + string(action.Kind), Trigger: filter("email"), Actions: []igdomain.WorkflowAction{action}, Now: f.now}); !errors.Is(err, usecases.ErrLifecycleWorkflowInvalidReference) {
			t.Fatalf("create without a %s repository = %v, want it refused", action.Kind, err)
		}
	}
}
