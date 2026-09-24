package usecases_test

// 主要ユースケース追跡: REQ-PROVISIONING-006。

import (
	"context"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

type fakeEnqueuer struct {
	calls   []struct{ tenantID, dedupKey, taskID string }
	nextJob string
	err     error
}

func (f *fakeEnqueuer) EnqueueProvisioningTask(_ context.Context, tenantID, dedupKey, taskID string) (string, error) {
	f.calls = append(f.calls, struct{ tenantID, dedupKey, taskID string }{tenantID, dedupKey, taskID})
	if f.err != nil {
		return "", f.err
	}
	return f.nextJob, nil
}

func TestDispatchPendingTasks_AttachesJobToEachUnenqueuedTask(t *testing.T) {
	taskRepo := memory.NewProvisioningTaskRepository()
	ctx := context.Background()
	d := &domain.ProvisioningTask{ID: "task-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.TaskPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := taskRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	enqueuer := &fakeEnqueuer{nextJob: "job-1"}
	dispatched, err := usecases.DispatchPendingTasks(ctx, usecases.DispatcherDeps{TaskRepo: taskRepo, Enqueuer: enqueuer}, 10, time.Now())
	if err != nil {
		t.Fatalf("DispatchPendingTasks() error = %v", err)
	}
	if dispatched != 1 {
		t.Errorf("DispatchPendingTasks() dispatched = %d, want 1", dispatched)
	}
	if len(enqueuer.calls) != 1 || enqueuer.calls[0].taskID != "task-1" || enqueuer.calls[0].dedupKey != d.IdempotencyKey() {
		t.Errorf("enqueuer.calls = %+v, want a single call for task-1 with dedupKey %q", enqueuer.calls, d.IdempotencyKey())
	}
	found, _ := taskRepo.Find(ctx, "tenant-a", "task-1")
	if found.JobID == nil || *found.JobID != "job-1" {
		t.Errorf("task.JobID = %v, want job-1", found.JobID)
	}
}

func TestDispatchPendingTasks_SkipsAlreadyAttached(t *testing.T) {
	taskRepo := memory.NewProvisioningTaskRepository()
	ctx := context.Background()
	d := &domain.ProvisioningTask{ID: "task-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.TaskPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	_, _ = taskRepo.Save(ctx, d)
	_, _ = taskRepo.AttachJob(ctx, "tenant-a", "task-1", "job-existing")

	enqueuer := &fakeEnqueuer{nextJob: "job-new"}
	dispatched, err := usecases.DispatchPendingTasks(ctx, usecases.DispatcherDeps{TaskRepo: taskRepo, Enqueuer: enqueuer}, 10, time.Now())
	if err != nil {
		t.Fatalf("DispatchPendingTasks() error = %v", err)
	}
	if dispatched != 0 || len(enqueuer.calls) != 0 {
		t.Errorf("DispatchPendingTasks() dispatched = %d, calls = %d, want 0 (already attached, not in ListUnenqueued)", dispatched, len(enqueuer.calls))
	}
}

//spec:covers EX-PROVISIONING-017-01: 未関連付けの pending プロビジョニングタスクを再走査してジョブを関連付け、プロビジョニングタスクを in_flight にして ProvisioningTaskStarted を発行する。
func TestDispatchPendingTasks_StartsTheTaskAndEmitsStartedAfterAttaching(t *testing.T) {
	taskRepo := memory.NewProvisioningTaskRepository()
	ctx := context.Background()
	d := &domain.ProvisioningTask{ID: "task-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.TaskPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := taskRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	recorder := &eventRecorder{}
	deps := usecases.DispatcherDeps{TaskRepo: taskRepo, Enqueuer: &fakeEnqueuer{nextJob: "job-1"}, Emit: recorder.emit}

	if _, err := usecases.DispatchPendingTasks(ctx, deps, 10, time.Now()); err != nil {
		t.Fatalf("DispatchPendingTasks() error = %v", err)
	}
	found, _ := taskRepo.Find(ctx, "tenant-a", "task-1")
	started, ok := onlyEvent[*domain.ProvisioningTaskStarted](t, recorder)
	if !ok || found.Status != domain.TaskInFlight ||
		started.TaskID != "task-1" || started.JobID != "job-1" || started.ConnectionID != "app-1" || started.TenantID != "tenant-a" {
		t.Fatalf("after dispatch: status = %v, event = %+v; want in_flight and one Started for task-1/job-1", found.Status, started)
	}
}

// 別の worker が同じプロビジョニングタスクへ先にジョブを関連付けた場合、関連付けに負けた側は発行しない。
func TestDispatchPendingTasks_EmitsNothingWhenAnotherWorkerAttachedFirst(t *testing.T) {
	taskRepo := memory.NewProvisioningTaskRepository()
	ctx := context.Background()
	d := &domain.ProvisioningTask{ID: "task-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1", SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.TaskPending, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if _, err := taskRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	recorder := &eventRecorder{}
	racing := racingEnqueuer{attach: func() { _, _ = taskRepo.AttachJob(ctx, "tenant-a", "task-1", "job-1") }}
	deps := usecases.DispatcherDeps{TaskRepo: taskRepo, Enqueuer: racing, Emit: recorder.emit}

	if _, err := usecases.DispatchPendingTasks(ctx, deps, 10, time.Now()); err != nil {
		t.Fatalf("DispatchPendingTasks() error = %v", err)
	}
	if len(recorder.events) != 0 {
		t.Fatalf("events = %v, want none from the worker that lost the attach", recorder.types())
	}
}

// racingEnqueuer は投入の間に別の worker の関連付けを割り込ませる。
type racingEnqueuer struct{ attach func() }

func (r racingEnqueuer) EnqueueProvisioningTask(context.Context, string, string, string) (string, error) {
	r.attach()
	return "job-1", nil
}

//spec:covers EX-PROVISIONING-006-01: 予約は期限の直前にはプロビジョニングタスクへ変わらず、期限に達した周期処理で delete のプロビジョニングタスクになってジョブへ関連付けられ、以後の周期処理では重複しない。
func TestDispatchPendingTasks_MaterializesAScheduledDeprovisionOnlyOnceDue(t *testing.T) {
	taskRepo := memory.NewProvisioningTaskRepository()
	ctx := context.Background()
	deletedAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	reservation := domain.NewScheduledDeprovision("reservation-1", "tenant-a", "app-1", "user-1", 42, deletedAt, 7)
	if _, err := taskRepo.ScheduleDeprovision(ctx, reservation); err != nil {
		t.Fatalf("ScheduleDeprovision() error = %v", err)
	}
	other := domain.NewScheduledDeprovision("reservation-2", "tenant-a", "app-2", "user-1", 42, deletedAt, 7)
	if _, err := taskRepo.ScheduleDeprovision(ctx, other); err != nil {
		t.Fatalf("ScheduleDeprovision(other) error = %v", err)
	}
	enqueuer := &fakeEnqueuer{nextJob: "job-1"}
	deps := usecases.DispatcherDeps{TaskRepo: taskRepo, Enqueuer: enqueuer}

	if _, err := usecases.DispatchPendingTasks(ctx, deps, 10, reservation.DueAt.Add(-time.Second)); err != nil {
		t.Fatalf("DispatchPendingTasks(before due) error = %v", err)
	}
	if tasks, _ := taskRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10); len(tasks) != 0 || len(enqueuer.calls) != 0 {
		t.Fatalf("before due: tasks = %+v, enqueues = %+v; want neither", tasks, enqueuer.calls)
	}

	if _, err := usecases.DispatchPendingTasks(ctx, deps, 10, reservation.DueAt); err != nil {
		t.Fatalf("DispatchPendingTasks(at due) error = %v", err)
	}
	tasks, _ := taskRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if len(tasks) != 1 {
		t.Fatalf("at due: tasks = %+v, want one", tasks)
	}
	d := tasks[0]
	if d.Operation != domain.OperationDelete || d.SourceID != "user-1" || d.SourceVersion != 42 || d.Status != domain.TaskInFlight {
		t.Errorf("materialized task = %+v, want an in-flight delete of user-1 at version 42", d)
	}
	if others, _ := taskRepo.ListByConnection(ctx, "tenant-a", "app-2", nil, 10); len(others) != 1 {
		t.Errorf("at due: tasks on app-2 = %+v, want every due reservation materialized in one pass", others)
	}
	if len(enqueuer.calls) != 2 {
		t.Errorf("enqueuer.calls = %+v, want one call per materialized task", enqueuer.calls)
	}

	if _, err := usecases.DispatchPendingTasks(ctx, deps, 10, reservation.DueAt.Add(time.Hour)); err != nil {
		t.Fatalf("DispatchPendingTasks(after due) error = %v", err)
	}
	if tasks, _ := taskRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10); len(tasks) != 1 || len(enqueuer.calls) != 2 {
		t.Errorf("after due: tasks = %d, enqueues = %d; want the one task only", len(tasks), len(enqueuer.calls))
	}
}
