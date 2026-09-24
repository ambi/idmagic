package usecases_test

// 主要ユースケース追跡: REQ-PROVISIONING-003、REQ-PROVISIONING-010、REQ-PROVISIONING-018。

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"testing"
	"time"

	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func newJobHandlerDeps(client *fakeTargetClient, attrSource *fakeAttributeSource) (usecases.JobHandlerDeps, *memory.ProvisioningConnectionRepository, *memory.ProvisioningTaskRepository) {
	executeTaskDeps, connRepo, taskRepo, _ := newExecuteTaskDeps(client, attrSource)
	return usecases.JobHandlerDeps{
		ExecuteTaskDeps: executeTaskDeps,
		ConnectionRepo:  connRepo,
		TaskRepo:        taskRepo,
		Now:             func() time.Time { return time.Now().UTC() },
	}, connRepo, taskRepo
}

const testJobMaxAttempts = 8

func newTestJob(t *testing.T, taskID string, attempts int) *jobsdomain.Job {
	t.Helper()
	params, err := json.Marshal(map[string]string{"task_id": taskID})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return &jobsdomain.Job{ID: "job-1", TenantID: testTenantID, Kind: usecases.KindProvisioningTask, Params: params, Attempts: attempts, MaxAttempts: testJobMaxAttempts}
}

func TestProvisioningTaskHandler_SuccessReturnsNilAndResetsFailureCount(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
	conn, _ := connRepo.Find(context.Background(), testTenantID, "app-1")
	conn.ConsecutiveFailureCount = 3
	_ = connRepo.Update(context.Background(), conn, nil)

	handler := usecases.ProvisioningTaskHandler(deps)
	_, err := handler(context.Background(), newTestJob(t, d.ID, 1))
	if err != nil {
		t.Fatalf("handler() error = %v", err)
	}
	got, _ := connRepo.Find(context.Background(), testTenantID, "app-1")
	if got.ConsecutiveFailureCount != 0 {
		t.Errorf("ConsecutiveFailureCount = %d, want 0 after success", got.ConsecutiveFailureCount)
	}
}

func TestProvisioningTaskHandler_NonTerminalFailureLeavesTaskInFlight(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)

	handler := usecases.ProvisioningTaskHandler(deps)
	_, err := handler(context.Background(), newTestJob(t, d.ID, 1))
	if err == nil {
		t.Fatal("handler() should return the downstream error so Jobs retries")
	}
	got, _ := taskRepo.Find(context.Background(), testTenantID, d.ID)
	if got.Status != domain.TaskInFlight {
		t.Errorf("task.Status = %v, want in_flight (non-terminal attempt)", got.Status)
	}
}

func TestProvisioningTaskHandler_TerminalFailureMarksDeadLetter(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)

	handler := usecases.ProvisioningTaskHandler(deps)
	_, err := handler(context.Background(), newTestJob(t, d.ID, 8)) // attempts == max_attempts: terminal
	if err == nil {
		t.Fatal("handler() should still return the error (Jobs itself records JobFailed terminal)")
	}
	got, _ := taskRepo.Find(context.Background(), testTenantID, d.ID)
	if got.Status != domain.TaskDeadLetter {
		t.Errorf("task.Status = %v, want dead_letter (terminal attempt)", got.Status)
	}
}

func TestProvisioningTaskHandler_QuarantinesConnectionAfterConsecutiveFailureThreshold(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	conn.QuarantineAfterConsecutiveFailure = 2
	if err := connRepo.Register(context.Background(), conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	handler := usecases.ProvisioningTaskHandler(deps)

	for i := range 2 {
		d := &domain.ProvisioningTask{
			ID: idFor(i), TenantID: testTenantID, ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
			SourceVersion: int64(i + 1), Operation: domain.OperationCreate, Status: domain.TaskInFlight, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if _, err := taskRepo.Save(context.Background(), d); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if _, err := handler(context.Background(), newTestJob(t, d.ID, 8)); err == nil {
			t.Fatal("handler() should return the downstream error")
		}
	}
	got, err := connRepo.Find(context.Background(), testTenantID, "app-1")
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if got.Health != domain.HealthQuarantined {
		t.Errorf("connection.Health = %v, want quarantined after %d consecutive terminal failures", got.Health, conn.QuarantineAfterConsecutiveFailure)
	}
}

func idFor(i int) string {
	return []string{"task-a", "task-b", "task-c"}[i]
}

func someRetryableErr() error { return &retryableTestErr{} }

type retryableTestErr struct{}

func (*retryableTestErr) Error() string { return "downstream unavailable" }

// emittedWith は発行の時点で保存先から読める状態を、イベントと対にして残す。
// 発行が保存より先に動けば、ここに保存前の状態が残る。
type emittedWith struct {
	event      spec.DomainEvent
	taskStatus domain.ProvisioningTaskStatus
	health     domain.ProvisioningHealth
}

func recordEmissions(deps *usecases.JobHandlerDeps, connRepo *memory.ProvisioningConnectionRepository, taskRepo *memory.ProvisioningTaskRepository) *[]emittedWith {
	var emitted []emittedWith
	deps.Emit = func(event spec.DomainEvent) {
		record := emittedWith{event: event}
		var taskID string
		switch e := event.(type) {
		case *domain.UserProvisioned:
			taskID = e.TaskID
		case *domain.UserProvisioningFailed:
			taskID = e.TaskID
		}
		if taskID != "" {
			d, _ := taskRepo.Find(context.Background(), testTenantID, taskID)
			record.taskStatus = d.Status
		}
		conn, _ := connRepo.Find(context.Background(), testTenantID, "app-1")
		record.health = conn.Health
		emitted = append(emitted, record)
	}
	return &emitted
}

//spec:covers EX-PROVISIONING-003-01: ジョブが下流への作成に成功すると、プロビジョニングタスクを succeeded に保存した後で UserProvisioned を一度だけ発行する。
func TestProvisioningTaskHandler_EmitsTheTransitionEventOnceAfterSucceeding(t *testing.T) {
	client := &fakeTargetClient{createUserID: "remote-1"}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
	emitted := recordEmissions(&deps, connRepo, taskRepo)
	handler := usecases.ProvisioningTaskHandler(deps)

	// Jobs はリースを失ったジョブを別の worker で再実行し得る。
	for range 2 {
		if _, err := handler(context.Background(), newTestJob(t, d.ID, 1)); err != nil {
			t.Fatalf("handler() error = %v", err)
		}
	}
	if len(*emitted) != 1 || client.createCalls != 1 {
		t.Fatalf("emitted = %+v, createCalls = %d; want one event and one downstream create across two runs", *emitted, client.createCalls)
	}
	got := (*emitted)[0]
	provisioned, ok := got.event.(*domain.UserProvisioned)
	if !ok || provisioned.RemoteID != "remote-1" || provisioned.TaskID != d.ID || got.taskStatus != domain.TaskSucceeded {
		t.Fatalf("emitted %+v with task status %v, want UserProvisioned for %s/remote-1 after succeeded", got.event, got.taskStatus, d.ID)
	}
}

//spec:covers EX-PROVISIONING-010-01: 最後の試行が失敗すると dead_letter を保存した後で UserProvisioningFailed を発行し、非終端の失敗では発行しない。
func TestProvisioningTaskHandler_TerminalFailureEmitsFailedThenQuarantinedOnce(t *testing.T) {
	client := &fakeTargetClient{createUserErr: someRetryableErr()}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	conn.QuarantineAfterConsecutiveFailure = 2
	if err := connRepo.Register(context.Background(), conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	emitted := recordEmissions(&deps, connRepo, taskRepo)
	handler := usecases.ProvisioningTaskHandler(deps)

	for i := range 3 {
		d := &domain.ProvisioningTask{
			ID: idFor(i), TenantID: testTenantID, ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
			SourceVersion: int64(i + 1), Operation: domain.OperationCreate, Status: domain.TaskInFlight, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if _, err := taskRepo.Save(context.Background(), d); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		if _, err := handler(context.Background(), newTestJob(t, d.ID, 1)); err == nil {
			t.Fatal("handler() non-terminal attempt should return the downstream error")
		}
		if _, err := handler(context.Background(), newTestJob(t, d.ID, testJobMaxAttempts)); err == nil {
			t.Fatal("handler() terminal attempt should return the downstream error")
		}
	}

	wantTypes := []string{"UserProvisioningFailed", "UserProvisioningFailed", "ConnectionQuarantined", "UserProvisioningFailed"}
	gotTypes := make([]string, 0, len(*emitted))
	for _, e := range *emitted {
		gotTypes = append(gotTypes, e.event.EventType())
	}
	if !slices.Equal(gotTypes, wantTypes) {
		t.Fatalf("emitted = %v, want %v", gotTypes, wantTypes)
	}
	failed := (*emitted)[0]
	if e := failed.event.(*domain.UserProvisioningFailed); e.TaskID != idFor(0) || e.SourceType != domain.SourceTypeUser ||
		e.SourceID != "user-1" || e.Error != "downstream unavailable" || failed.taskStatus != domain.TaskDeadLetter {
		t.Errorf("first failure = %+v with task status %v, want task-a/user-1 after dead_letter", e, failed.taskStatus)
	}
	quarantined := (*emitted)[2]
	if e := quarantined.event.(*domain.ConnectionQuarantined); e.ApplicationID != "app-1" || e.ConsecutiveFailures != 2 ||
		e.Reason != "downstream unavailable" || quarantined.health != domain.HealthQuarantined {
		t.Errorf("quarantine = %+v with health %v, want app-1 after 2 failures once quarantined", e, quarantined.health)
	}
}

//spec:covers EX-PROVISIONING-018-01: 必須属性を解決できないプロビジョニングタスクは、試行回数が残っていても dead_letter になり UserProvisioningFailed を発行する。
func TestProvisioningTaskHandler_UnresolvedRequiredMappingDeadLettersOnFirstAttempt(t *testing.T) {
	client := &fakeTargetClient{createUserErr: fmt.Errorf("mapping emails: %w", ports.ErrRequiredAttributeUnresolved)}
	attrSource := &fakeAttributeSource{attrs: map[string]any{"preferred_username": "alice"}, exists: true}
	deps, connRepo, taskRepo := newJobHandlerDeps(client, attrSource)
	d := setupConnectionAndTask(t, connRepo, taskRepo, domain.OperationCreate)
	emitted := recordEmissions(&deps, connRepo, taskRepo)

	if _, err := usecases.ProvisioningTaskHandler(deps)(context.Background(), newTestJob(t, d.ID, 1)); err != nil {
		t.Fatalf("handler() error = %v, want nil so Jobs does not retry a settled task", err)
	}
	got, _ := taskRepo.Find(context.Background(), testTenantID, d.ID)
	conn, _ := connRepo.Find(context.Background(), testTenantID, "app-1")
	if got.Status != domain.TaskDeadLetter || len(*emitted) != 1 || (*emitted)[0].event.EventType() != "UserProvisioningFailed" {
		t.Fatalf("status = %v, emitted = %+v; want dead_letter and one UserProvisioningFailed on the first attempt", got.Status, *emitted)
	}
	// 属性の欠落は 1 人の User の問題であり、下流の健全性を表さない。
	if conn.ConsecutiveFailureCount != 0 {
		t.Errorf("ConsecutiveFailureCount = %d, want 0", conn.ConsecutiveFailureCount)
	}
}
