package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/provisioning/internal/usecases"
)

type quarantineTaskRepository struct {
	ports.ProvisioningTaskRepository
	suspendError error
	resyncs      int
}

func (r *quarantineTaskRepository) SuspendForQuarantine(ctx context.Context, tenantID, taskID, jobID string) (bool, error) {
	if r.suspendError != nil {
		return false, r.suspendError
	}
	return r.ProvisioningTaskRepository.SuspendForQuarantine(ctx, tenantID, taskID, jobID)
}

func (r *quarantineTaskRepository) SaveFullResync(ctx context.Context, resync *domain.FullResync) error {
	r.resyncs++
	return r.ProvisioningTaskRepository.SaveFullResync(ctx, resync)
}

//spec:covers REQ-PROVISIONING-020: 隔離中は操作と主体の種類にかかわらず、資格情報を使うクライアントの構築前に実行を止める。
func TestExecuteTask_QuarantineStopsEveryOperationBeforeClient(t *testing.T) {
	for _, operation := range []domain.ProvisioningOperation{domain.OperationCreate, domain.OperationUpdate, domain.OperationDeactivate, domain.OperationDelete, domain.OperationMembershipAdd, domain.OperationMembershipRemove} {
		t.Run(string(operation), func(t *testing.T) {
			ctx := t.Context()
			deps, connections, tasks, _ := newExecuteTaskDeps(&fakeTargetClient{}, &fakeAttributeSource{exists: true})
			task := setupConnectionAndTask(t, connections, tasks, operation)
			if operation == domain.OperationMembershipAdd || operation == domain.OperationMembershipRemove {
				groupTask := *task
				groupTask.ID, groupTask.SourceType = "group-task", domain.SourceTypeGroup
				if _, err := tasks.Save(ctx, &groupTask); err != nil {
					t.Fatal(err)
				}
				task = &groupTask
			}
			conn, err := connections.Find(ctx, testTenantID, task.ConnectionID)
			if err != nil {
				t.Fatal(err)
			}
			if err := conn.Quarantine("unavailable", time.Now()); err != nil {
				t.Fatal(err)
			}
			if err := connections.Update(ctx, conn, nil); err != nil {
				t.Fatal(err)
			}
			built := false
			deps.NewTargetClient = func(*domain.ProvisioningConnection, string) (ports.ProvisioningTargetClient, error) {
				built = true
				return &fakeTargetClient{}, nil
			}
			event, execErr := usecases.ExecuteTask(ctx, deps, testTenantID, task.ID, time.Now())
			stored, err := tasks.Find(ctx, testTenantID, task.ID)
			if err != nil || execErr == nil || event != nil || built || stored.Status != domain.TaskInFlight {
				t.Fatalf("event=%v error=%v built=%v task=%+v read=%v; want stopped before client construction", event, execErr, built, stored, err)
			}
		})
	}
}

//spec:covers REQ-PROVISIONING-012, REQ-PROVISIONING-013: 隔離は対象の列挙より先に拒否され、空のフル同期、タスクと完了イベントを作らない。
func TestManualSynchronization_QuarantineCreatesNothing(t *testing.T) {
	ctx := t.Context()
	deps, connections, tasks := newAdminDeps()
	conn := activeConnection("app-1", domain.ScopeAllUsers)
	if err := conn.Quarantine("unavailable", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := connections.Register(ctx, conn, "secret"); err != nil {
		t.Fatal(err)
	}
	recorder := &eventRecorder{}
	deps.Emit = recorder.emit
	repository := &quarantineTaskRepository{ProvisioningTaskRepository: tasks}
	deps.TaskRepo = repository
	if _, err := usecases.StartFullResync(ctx, deps, testTenantID, "app-1", time.Now()); !errors.Is(err, usecases.ErrConnectionQuarantined) {
		t.Fatalf("resync error=%v; want quarantine before reading subjects", err)
	}
	if _, err := usecases.ProvisionOnDemand(ctx, deps, testTenantID, "app-1", domain.SourceTypeUser, "user-1", time.Now()); !errors.Is(err, usecases.ErrConnectionQuarantined) {
		t.Fatalf("on-demand error=%v; want quarantine", err)
	}
	stored, err := tasks.ListByConnection(ctx, testTenantID, "app-1", nil, 10)
	if err != nil || len(stored) != 0 || len(recorder.events) != 0 || repository.resyncs != 0 {
		t.Fatalf("tasks=%v resyncs=%d events=%v error=%v; want no synchronization effects", stored, repository.resyncs, recorder.types(), err)
	}
}

//spec:covers REQ-PROVISIONING-020: 保留の保存に失敗した場合はジョブへ失敗を返し、タスクを成功にも配信不能にもしない。
func TestProvisioningTaskHandler_QuarantineSaveFailureIsReturned(t *testing.T) {
	ctx := t.Context()
	client := &fakeTargetClient{}
	deps, connections, tasks := newJobHandlerDeps(client, &fakeAttributeSource{exists: true})
	task := setupConnectionAndTask(t, connections, tasks, domain.OperationDelete)
	conn, err := connections.Find(ctx, testTenantID, task.ConnectionID)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Quarantine("unavailable", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := connections.Update(ctx, conn, nil); err != nil {
		t.Fatal(err)
	}
	saveError := errors.New("storage unavailable")
	deps.TaskRepo = &quarantineTaskRepository{ProvisioningTaskRepository: tasks, suspendError: saveError}
	if _, err := usecases.ProvisioningTaskHandler(deps)(ctx, newTestJob(t, task.ID, testJobMaxAttempts)); !errors.Is(err, saveError) {
		t.Fatalf("handler error=%v; want suspension save error", err)
	}
	stored, err := tasks.Find(ctx, testTenantID, task.ID)
	if err != nil || stored.Status != domain.TaskInFlight || client.deleteCalls != 0 {
		t.Fatalf("task=%+v calls=%d error=%v; want untouched in_flight", stored, client.deleteCalls, err)
	}
}
