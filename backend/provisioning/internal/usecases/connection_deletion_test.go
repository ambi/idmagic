package usecases_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
	provisioningusecases "github.com/ambi/idmagic/backend/provisioning/internal/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-PROVISIONING-022: 消えた旧タスクのジョブは試行回数にかかわらず、新接続の資格情報と状態を使わず終了する。
func TestProvisioningTaskHandlerDeletedTaskDoesNotUseRecreatedConnection(t *testing.T) {
	for _, attempts := range []int{1, testJobMaxAttempts} {
		t.Run(fmt.Sprint(attempts), func(t *testing.T) {
			ctx := t.Context()
			deps, connections, tasks := newJobHandlerDeps(&fakeTargetClient{}, &fakeAttributeSource{exists: true})
			conn := activeConnection("app-1", provisioningdomain.ScopeAllUsers)
			conn.ConsecutiveFailureCount = 3
			if err := connections.Register(ctx, conn, "new-secret"); err != nil {
				t.Fatal(err)
			}
			newTask := &provisioningdomain.ProvisioningTask{
				ID: "new-task", TenantID: testTenantID, ConnectionID: "app-1", SourceType: provisioningdomain.SourceTypeUser,
				SourceID: "user-1", SourceVersion: 1, Operation: provisioningdomain.OperationCreate,
				Status: provisioningdomain.TaskPending, CreatedAt: conn.CreatedAt, UpdatedAt: conn.UpdatedAt,
			}
			resync := provisioningdomain.NewFullResync("new-resync", testTenantID, "app-1", 1, conn.CreatedAt)
			if err := tasks.SaveFullResync(ctx, resync); err != nil {
				t.Fatal(err)
			}
			if _, err := tasks.SaveFullResyncTask(ctx, resync.ID, newTask); err != nil {
				t.Fatal(err)
			}
			credentials := &credentialReadRecorder{ProvisioningConnectionRepository: connections}
			deps.ExecuteTaskDeps.ConnectionRepo = credentials
			deps.ExecuteTaskDeps.NewTargetClient = func(*provisioningdomain.ProvisioningConnection, string) (provisioningports.ProvisioningTargetClient, error) {
				t.Error("deleted task built a downstream client")
				return nil, errors.New("unexpected downstream client")
			}
			var events []spec.DomainEvent
			deps.Emit = func(event spec.DomainEvent) { events = append(events, event) }
			if _, err := provisioningusecases.ProvisioningTaskHandler(deps)(ctx, newTestJob(t, "deleted-task", attempts)); err != nil {
				t.Fatalf("deleted job error=%v; want normal completion", err)
			}
			storedConn, err := connections.Find(ctx, testTenantID, "app-1")
			if err != nil || !reflect.DeepEqual(storedConn, conn) {
				t.Fatalf("connection=%+v error=%v", storedConn, err)
			}
			storedTask, err := tasks.Find(ctx, testTenantID, newTask.ID)
			if err != nil || !reflect.DeepEqual(storedTask, newTask) {
				t.Fatalf("task=%+v error=%v", storedTask, err)
			}
			storedResync, err := tasks.FindFullResyncByTask(ctx, testTenantID, newTask.ID)
			if err != nil || !reflect.DeepEqual(storedResync, resync) {
				t.Fatalf("resync=%+v error=%v", storedResync, err)
			}
			if credentials.reads != 0 || len(events) != 0 {
				t.Fatalf("credential reads=%d events=%v; want none", credentials.reads, events)
			}
		})
	}
}

type deletionErrorRepository struct {
	provisioningports.ProvisioningConnectionRepository
	err error
}

func (r deletionErrorRepository) Delete(context.Context, string, string) error { return r.err }

//spec:covers REQ-PROVISIONING-021, REQ-PROVISIONING-015: 削除を保存できないか別テナントから要求した場合、接続とタスクを変えずイベントも発行しない。
func TestDeleteConnectionFailureLeavesStateAndEventsUntouched(t *testing.T) {
	ctx := t.Context()
	deps, connections, tasks := newAdminDeps()
	conn := activeConnection("app-1", provisioningdomain.ScopeAllUsers)
	if err := connections.Register(ctx, conn, "secret"); err != nil {
		t.Fatal(err)
	}
	task := &provisioningdomain.ProvisioningTask{
		ID: "task-1", TenantID: testTenantID, ConnectionID: "app-1", SourceType: provisioningdomain.SourceTypeUser,
		SourceID: "user-1", SourceVersion: 1, Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.TaskPending,
		CreatedAt: conn.CreatedAt, UpdatedAt: conn.UpdatedAt,
	}
	if _, err := tasks.Save(ctx, task); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	deps.Emit = func(event spec.DomainEvent) { events = append(events, event) }
	if err := provisioningusecases.DeleteConnection(ctx, deps, "other-tenant", "app-1", time.Now()); !errors.Is(err, provisioningusecases.ErrConnectionNotFound) {
		t.Fatalf("other tenant delete=%v", err)
	}
	injected := errors.New("delete failed")
	deps.ConnectionRepo = deletionErrorRepository{ProvisioningConnectionRepository: connections, err: injected}
	if err := provisioningusecases.DeleteConnection(ctx, deps, testTenantID, "app-1", time.Now()); !errors.Is(err, injected) {
		t.Fatalf("delete error=%v", err)
	}
	stored, err := connections.Find(ctx, testTenantID, "app-1")
	if err != nil || !reflect.DeepEqual(stored, conn) {
		t.Fatalf("connection=%+v error=%v", stored, err)
	}
	storedTask, err := tasks.Find(ctx, testTenantID, task.ID)
	if err != nil || !reflect.DeepEqual(storedTask, task) || len(events) != 0 {
		t.Fatalf("task=%+v events=%v error=%v", storedTask, events, err)
	}
}

type taskReadFailure struct {
	provisioningports.ProvisioningTaskRepository
	err error
}

func (r taskReadFailure) Find(context.Context, string, string) (*provisioningdomain.ProvisioningTask, error) {
	return nil, r.err
}

//spec:covers REQ-PROVISIONING-022: 保存先の障害を削除済みと見なしてジョブを完了させない。
func TestProvisioningTaskHandlerTaskReadFailureIsRetryable(t *testing.T) {
	deps, _, tasks := newJobHandlerDeps(&fakeTargetClient{}, &fakeAttributeSource{exists: true})
	injected := errors.New("read failed")
	deps.TaskRepo = taskReadFailure{ProvisioningTaskRepository: tasks, err: injected}
	if _, err := provisioningusecases.ProvisioningTaskHandler(deps)(t.Context(), newTestJob(t, "task-1", 1)); !errors.Is(err, injected) {
		t.Fatalf("job error=%v; want read failure", err)
	}
}

type deleteAfterTaskRead struct {
	provisioningports.ProvisioningTaskRepository
	remove func() error
}

func (r deleteAfterTaskRead) Find(ctx context.Context, tenantID, taskID string) (*provisioningdomain.ProvisioningTask, error) {
	task, err := r.ProvisioningTaskRepository.Find(ctx, tenantID, taskID)
	if err != nil {
		return nil, err
	}
	if err := r.remove(); err != nil {
		return nil, err
	}
	return task, nil
}

//spec:covers REQ-PROVISIONING-022: ジョブが旧タスクを読んだ直後に削除が確定しても、実行側のタスク消失を正常終了として扱う。
func TestProvisioningTaskHandlerDeletionBetweenTaskLookups(t *testing.T) {
	ctx := t.Context()
	deps, connections, tasks := newJobHandlerDeps(&fakeTargetClient{}, &fakeAttributeSource{exists: true})
	task := setupConnectionAndTask(t, connections, tasks, provisioningdomain.OperationCreate)
	deps.TaskRepo = deleteAfterTaskRead{ProvisioningTaskRepository: tasks, remove: func() error {
		return connections.Delete(ctx, testTenantID, task.ConnectionID)
	}}
	credentials := &credentialReadRecorder{ProvisioningConnectionRepository: connections}
	deps.ExecuteTaskDeps.ConnectionRepo = credentials
	var events []spec.DomainEvent
	deps.Emit = func(e spec.DomainEvent) { events = append(events, e) }
	if _, err := provisioningusecases.ProvisioningTaskHandler(deps)(ctx, newTestJob(t, task.ID, testJobMaxAttempts)); err != nil {
		t.Fatalf("job error=%v", err)
	}
	if credentials.reads != 0 || len(events) != 0 {
		t.Fatalf("credential reads=%d events=%v; want none", credentials.reads, events)
	}
}
