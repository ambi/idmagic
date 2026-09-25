package usecases_test

// 主要ユースケース追跡: REQ-PROVISIONING-013。

import (
	"context"
	"testing"
	"time"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// fullResyncFixture は all_users の接続と User を置き、StartFullResync とジョブハンドラーの発行を一つに記録する。
type fullResyncFixture struct {
	t        *testing.T
	admin    usecases.AdminDeps
	handler  usecases.JobHandlerDeps
	client   *fakeTargetClient
	taskRepo *memory.ProvisioningTaskRepository
	events   []spec.DomainEvent
}

func newFullResyncFixture(t *testing.T, userIDs ...string) *fullResyncFixture {
	t.Helper()
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "someone"}, exists: true}
	handlerDeps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	now := time.Now().UTC()
	conn := &domain.ProvisioningConnection{
		ApplicationID: "app-1", TenantID: testTenantID, Status: domain.ConnectionActive,
		BaseURL:      "https://downstream.example.com/scim/v2",
		Credential:   domain.ProvisioningConnectionCredentialMetadata{CredentialID: "cred", AuthMethod: domain.AuthBearerToken, CreatedAt: now},
		FeatureFlags: domain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true},
		Scope:        domain.ScopeAllUsers,
		Matching:     domain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy: domain.DeprovisionPolicy{
			OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate,
		},
		RateLimitPerMinute: 60, MaxAttempts: testJobMaxAttempts, QuarantineAfterConsecutiveFailure: 10,
		Health: domain.HealthOK, CreatedAt: now, UpdatedAt: now,
	}
	if err := connRepo.Register(context.Background(), conn, "tok"); err != nil {
		t.Fatal(err)
	}
	users := usermemory.NewUserRepository()
	for _, id := range userIDs {
		users.Seed(&userdomain.User{ID: id, TenantID: testTenantID, PreferredUsername: id, CreatedAt: now, UpdatedAt: now})
	}
	f := &fullResyncFixture{t: t, handler: handlerDeps, client: client, taskRepo: taskRepo}
	f.admin = usecases.AdminDeps{ConnectionRepo: connRepo, TaskRepo: taskRepo, UserRepo: users, Emit: f.record}
	f.handler.Emit = f.record
	return f
}

func (f *fullResyncFixture) record(event spec.DomainEvent) { f.events = append(f.events, event) }

func (f *fullResyncFixture) completed() []*domain.FullResyncCompleted {
	var out []*domain.FullResyncCompleted
	for _, event := range f.events {
		if c, ok := event.(*domain.FullResyncCompleted); ok {
			out = append(out, c)
		}
	}
	return out
}

func (f *fullResyncFixture) start() []*domain.ProvisioningTask {
	f.t.Helper()
	if _, err := usecases.StartFullResync(context.Background(), f.admin, testTenantID, "app-1", time.Now().UTC()); err != nil {
		f.t.Fatalf("StartFullResync() error = %v", err)
	}
	tasks, err := f.taskRepo.ListByConnection(context.Background(), testTenantID, "app-1", nil, 50)
	if err != nil {
		f.t.Fatal(err)
	}
	return tasks
}

func (f *fullResyncFixture) run(task *domain.ProvisioningTask, attempts int) error {
	f.t.Helper()
	_, err := usecases.ProvisioningTaskHandler(f.handler)(context.Background(), newTestJob(f.t, task.ID, attempts))
	return err
}

//spec:covers EX-PROVISIONING-013-01: 成功と失敗が混じっても、最後のプロビジョニングタスクが終端になったときだけ FullResyncCompleted を一度発行し、再実行では増やさない。
func TestFullResyncCompletesOnceWhenTheLastTaskSettles(t *testing.T) {
	f := newFullResyncFixture(t, "u1", "u2", "u3")
	tasks := f.start()
	if len(tasks) != 3 || len(f.completed()) != 0 {
		t.Fatalf("after start: tasks = %d, completed = %d; want 3 pending tasks and no completion", len(tasks), len(f.completed()))
	}

	if err := f.run(tasks[0], 1); err != nil {
		t.Fatalf("run(success) error = %v", err)
	}
	f.client.createUserErr = someRetryableErr()
	if err := f.run(tasks[1], 1); err == nil {
		t.Fatal("run(non-terminal failure) error = nil, want the downstream error")
	}
	f.client.createUserErr = nil
	if err := f.run(tasks[2], 1); err != nil {
		t.Fatalf("run(success) error = %v", err)
	}
	if got := f.completed(); len(got) != 0 {
		t.Fatalf("completed while a task is still in flight = %+v", got)
	}
	// 最後に終端になるのは再試行を使い切った失敗である。
	f.client.createUserErr = someRetryableErr()
	if err := f.run(tasks[1], testJobMaxAttempts); err == nil {
		t.Fatal("run(terminal failure) error = nil, want the downstream error")
	}
	if got := f.completed(); len(got) != 1 {
		t.Fatalf("FullResyncCompleted after the last task dead-lettered = %d events, want one without waiting for a rerun", len(got))
	}
	// 同じジョブの再実行（リースの喪失など）は完了を二度発行しない。
	if err := f.run(tasks[1], testJobMaxAttempts); err != nil {
		t.Fatalf("run(dead-lettered, rerun) error = %v", err)
	}

	got := f.completed()
	if len(got) != 1 {
		t.Fatalf("FullResyncCompleted = %d events, want exactly one", len(got))
	}
	if got[0].TenantID != testTenantID || got[0].ApplicationID != "app-1" ||
		got[0].TotalSubjects != 3 || got[0].SucceededCount != 2 || got[0].FailedCount != 1 {
		t.Fatalf("FullResyncCompleted = %+v, want 3 subjects, 2 succeeded, 1 failed on app-1", *got[0])
	}
	if last := f.events[len(f.events)-1]; last != got[0] {
		t.Fatalf("last event = %s, want FullResyncCompleted after the task's own transition", last.EventType())
	}
}

//spec:covers EX-PROVISIONING-013-01: 明示指定に同じ Group が重複していても subject は一件と数え、そのプロビジョニングタスクの終端で完了する。
func TestFullResyncCountsADuplicatedSubjectOnce(t *testing.T) {
	f := newFullResyncFixture(t)
	conn, err := f.admin.ConnectionRepo.Find(context.Background(), testTenantID, "app-1")
	if err != nil || conn == nil {
		t.Fatalf("Find() = %+v, %v", conn, err)
	}
	conn.FeatureFlags.PushGroups = true
	conn.GroupPush = &domain.GroupPushConfig{Selection: domain.GroupSelectionExplicit, ExplicitGroupIDs: []string{"g1", "g1"}}
	if err := f.admin.ConnectionRepo.Update(context.Background(), conn, nil); err != nil {
		t.Fatal(err)
	}
	tasks := f.start()
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want one for g1", len(tasks))
	}
	if err := f.run(tasks[0], 1); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got := f.completed(); len(got) != 1 || got[0].TotalSubjects != 1 || got[0].SucceededCount != 1 {
		t.Fatalf("FullResyncCompleted = %v, want one with 1 subject succeeded", got)
	}
}

// staleFullResyncRepo は、別のジョブが完了させる前に読んだ running のフル同期を返し続ける。
// 最後の二件を同時に終えた二つのジョブが、どちらも running を読んで完了を判定する場面を決定的に再現する。
type staleFullResyncRepo struct {
	*memory.ProvisioningTaskRepository
	stale *domain.FullResync
}

func (r staleFullResyncRepo) FindFullResyncByTask(context.Context, string, string) (*domain.FullResync, error) {
	clone := *r.stale
	return &clone, nil
}

//spec:covers EX-PROVISIONING-013-01: 同時に完了を判定した二つのジョブがどちらも running を読んでも、FullResyncCompleted は一度だけ発行される。
func TestFullResyncConcurrentSettlementEmitsOnce(t *testing.T) {
	f := newFullResyncFixture(t, "u1")
	tasks := f.start()
	stale, err := f.taskRepo.FindFullResyncByTask(context.Background(), testTenantID, tasks[0].ID)
	if err != nil || stale == nil {
		t.Fatalf("FindFullResyncByTask() = %+v, %v", stale, err)
	}
	if err := f.run(tasks[0], 1); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	f.handler.TaskRepo = staleFullResyncRepo{ProvisioningTaskRepository: f.taskRepo, stale: stale}
	if err := f.run(tasks[0], 2); err != nil {
		t.Fatalf("run(with a stale running read) error = %v", err)
	}
	if got := f.completed(); len(got) != 1 {
		t.Fatalf("FullResyncCompleted = %d events, want exactly one", len(got))
	}
}

//spec:covers EX-PROVISIONING-013-01: scope 内に subject がなければ、開始と同時に対象 0 件の FullResyncCompleted を発行する。
func TestStartFullResyncCompletesAnEmptyScopeImmediately(t *testing.T) {
	f := newFullResyncFixture(t)
	if tasks := f.start(); len(tasks) != 0 {
		t.Fatalf("tasks = %d, want none", len(tasks))
	}
	got := f.completed()
	if len(got) != 1 || got[0].TotalSubjects != 0 || got[0].SucceededCount != 0 || got[0].FailedCount != 0 {
		t.Fatalf("FullResyncCompleted = %v, want one with zero subjects", got)
	}
}

//spec:covers EX-PROVISIONING-013-01: フル同期に属さないプロビジョニングタスクの終端化は完了イベントを発行しない。
func TestTaskOutsideAFullResyncDoesNotComplete(t *testing.T) {
	f := newFullResyncFixture(t, "u1")
	task := saveTask(t, f.taskRepo, domain.OperationCreate, 1)
	if err := f.run(task, 1); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if got := f.completed(); len(got) != 0 {
		t.Fatalf("FullResyncCompleted = %v, want none", got)
	}
}
