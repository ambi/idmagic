package usecases_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/provisioning/internal/db_memory"
	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
	provisioningusecases "github.com/ambi/idmagic/backend/provisioning/internal/usecases"
)

type credentialReadRecorder struct {
	provisioningports.ProvisioningConnectionRepository
	reads int
}

func (r *credentialReadRecorder) CredentialSecret(ctx context.Context, tenantID, applicationID string) (string, error) {
	r.reads++
	return r.ProvisioningConnectionRepository.CredentialSecret(ctx, tenantID, applicationID)
}

//spec:covers REQ-PROVISIONING-020: 無効化は主体と操作の種類にかかわらず資格情報の取得前に送信を止める。
func TestExecuteTask_DisabledStopsEveryOperationBeforeCredentials(t *testing.T) {
	for _, sourceType := range []provisioningdomain.ProvisioningSourceType{provisioningdomain.SourceTypeUser, provisioningdomain.SourceTypeGroup} {
		for _, operation := range []provisioningdomain.ProvisioningOperation{provisioningdomain.OperationCreate, provisioningdomain.OperationUpdate, provisioningdomain.OperationDeactivate, provisioningdomain.OperationDelete, provisioningdomain.OperationMembershipAdd, provisioningdomain.OperationMembershipRemove} {
			t.Run(string(sourceType)+"/"+string(operation), func(t *testing.T) {
				ctx := t.Context()
				deps, connections, tasks, _ := newExecuteTaskDeps(&fakeTargetClient{}, &fakeAttributeSource{exists: true})
				task := setupConnectionAndTask(t, connections, tasks, operation)
				if sourceType == provisioningdomain.SourceTypeGroup {
					groupTask := *task
					groupTask.ID, groupTask.SourceType = "group-task", sourceType
					if _, err := tasks.Save(ctx, &groupTask); err != nil {
						t.Fatal(err)
					}
					task = &groupTask
				}
				conn, err := connections.Find(ctx, testTenantID, task.ConnectionID)
				if err != nil {
					t.Fatal(err)
				}
				conn.Status = provisioningdomain.ConnectionDisabled
				if err := connections.Update(ctx, conn, nil); err != nil {
					t.Fatal(err)
				}
				credentials := &credentialReadRecorder{ProvisioningConnectionRepository: connections}
				deps.ConnectionRepo = credentials
				built := false
				deps.NewTargetClient = func(*provisioningdomain.ProvisioningConnection, string) (provisioningports.ProvisioningTargetClient, error) {
					built = true
					return nil, errors.New("unexpected client construction")
				}
				event, execErr := provisioningusecases.ExecuteTask(ctx, deps, testTenantID, task.ID, time.Now())
				stored, err := tasks.Find(ctx, testTenantID, task.ID)
				if err != nil || execErr == nil || event != nil || credentials.reads != 0 || built || stored.Status != provisioningdomain.TaskInFlight {
					t.Fatalf("event=%v error=%v reads=%d built=%v task=%+v read=%v", event, execErr, credentials.reads, built, stored, err)
				}
			})
		}
	}
}

//spec:covers REQ-PROVISIONING-017, REQ-PROVISIONING-020: 無効の接続が先頭のページを埋めても、正常な接続だけを投入し、無効の pending は変更しない。
func TestDispatchPendingTasks_SkipsDisabledWithoutStarvingHealthyTasks(t *testing.T) {
	ctx := t.Context()
	connections := dispatcherConnections(t)
	conn, err := connections.Find(ctx, testTenantID, "app-1")
	if err != nil {
		t.Fatal(err)
	}
	conn.Status = provisioningdomain.ConnectionDisabled
	if err := connections.Update(ctx, conn, nil); err != nil {
		t.Fatal(err)
	}
	tasks := memory.NewProvisioningTaskRepository()
	now := time.Now()
	for i, id := range []string{"held-1", "held-2", "healthy"} {
		connectionID := "app-1"
		if id == "healthy" {
			connectionID = "app-2"
		}
		if _, err := tasks.Save(ctx, &provisioningdomain.ProvisioningTask{
			ID: id, TenantID: testTenantID, ConnectionID: connectionID, SourceType: provisioningdomain.SourceTypeUser,
			SourceID: id, SourceVersion: 1, Operation: provisioningdomain.OperationDelete, Status: provisioningdomain.TaskPending,
			CreatedAt: now.Add(time.Duration(i) * time.Microsecond), UpdatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	enqueuer := &fakeEnqueuer{nextJob: "job-healthy"}
	count, err := provisioningusecases.DispatchPendingTasks(ctx, provisioningusecases.DispatcherDeps{ConnectionRepo: connections, TaskRepo: tasks, Enqueuer: enqueuer}, 1, now)
	if err != nil || count != 1 || len(enqueuer.calls) != 1 || enqueuer.calls[0].taskID != "healthy" {
		t.Fatalf("count=%d calls=%v error=%v", count, enqueuer.calls, err)
	}
	for _, id := range []string{"held-1", "held-2"} {
		stored, err := tasks.Find(ctx, testTenantID, id)
		if err != nil || stored.Status != provisioningdomain.TaskPending || stored.JobID != nil {
			t.Fatalf("held=%+v error=%v", stored, err)
		}
	}
}

//spec:covers REQ-PROVISIONING-020: 無効化で保留するタスクをフル同期の完了として数えず、失敗数や成功失敗のイベントも変えない。
func TestDisabledTaskDoesNotCompleteFullResync(t *testing.T) {
	f := newFullResyncFixture(t, "user-1")
	task := f.start()[0]
	ctx := t.Context()
	if attached, err := f.taskRepo.AttachJob(ctx, testTenantID, task.ID, "job-1", nil); err != nil || !attached {
		t.Fatalf("attach=%v error=%v", attached, err)
	}
	conn, err := f.admin.ConnectionRepo.Find(ctx, testTenantID, "app-1")
	if err != nil {
		t.Fatal(err)
	}
	conn.Status, conn.ConsecutiveFailureCount = provisioningdomain.ConnectionDisabled, 3
	beforeFullSyncAt := conn.LastFullSyncAt
	if err := f.admin.ConnectionRepo.Update(ctx, conn, nil); err != nil {
		t.Fatal(err)
	}
	before := len(f.events)
	if err := f.run(task, testJobMaxAttempts); err != nil {
		t.Fatal(err)
	}
	stored, err := f.taskRepo.Find(ctx, testTenantID, task.ID)
	if err != nil || stored.Status != provisioningdomain.TaskPending || stored.JobID == nil || *stored.JobID != "job-1" {
		t.Fatalf("held=%+v error=%v", stored, err)
	}
	if len(f.completed()) != 0 || len(f.events) != before || f.client.createCalls != 0 || f.client.updateCalls != 0 {
		t.Fatalf("events=%v client=%+v; want no completion or send", f.events, f.client)
	}
	conn, err = f.admin.ConnectionRepo.Find(ctx, testTenantID, "app-1")
	if err != nil || conn.ConsecutiveFailureCount != 3 || !reflect.DeepEqual(conn.LastFullSyncAt, beforeFullSyncAt) {
		t.Fatalf("connection=%+v error=%v", conn, err)
	}
}

//spec:covers REQ-PROVISIONING-020: 無効化による保留の保存失敗はジョブへ返し、実行中のタスクと接続を維持する。
func TestDisabledTaskSuspensionSaveFailureIsReturned(t *testing.T) {
	ctx := t.Context()
	client := &fakeTargetClient{}
	deps, connections, tasks := newJobHandlerDeps(client, &fakeAttributeSource{exists: true})
	task := setupConnectionAndTask(t, connections, tasks, provisioningdomain.OperationDelete)
	if err := tasks.UpdateStatus(ctx, testTenantID, task.ID, provisioningdomain.TaskPending, nil); err != nil {
		t.Fatal(err)
	}
	if attached, err := tasks.AttachJob(ctx, testTenantID, task.ID, "job-1", nil); err != nil || !attached {
		t.Fatalf("attach=%v error=%v", attached, err)
	}
	conn, err := connections.Find(ctx, testTenantID, task.ConnectionID)
	if err != nil {
		t.Fatal(err)
	}
	conn.Status, conn.ConsecutiveFailureCount = provisioningdomain.ConnectionDisabled, 3
	if err := connections.Update(ctx, conn, nil); err != nil {
		t.Fatal(err)
	}
	saveError := errors.New("storage unavailable")
	deps.TaskRepo = &quarantineTaskRepository{ProvisioningTaskRepository: tasks, suspendError: saveError}
	recorder := &eventRecorder{}
	deps.Emit = recorder.emit
	if _, err := provisioningusecases.ProvisioningTaskHandler(deps)(ctx, newTestJob(t, task.ID, testJobMaxAttempts)); !errors.Is(err, saveError) {
		t.Fatalf("handler error=%v; want suspension save error", err)
	}
	stored, err := tasks.Find(ctx, testTenantID, task.ID)
	if err != nil || stored.Status != provisioningdomain.TaskInFlight || client.deleteCalls != 0 || len(recorder.events) != 0 {
		t.Fatalf("task=%+v calls=%d error=%v events=%v", stored, client.deleteCalls, err, recorder.types())
	}
	conn, err = connections.Find(ctx, testTenantID, task.ConnectionID)
	if err != nil || conn.ConsecutiveFailureCount != 3 || conn.Status != provisioningdomain.ConnectionDisabled {
		t.Fatalf("connection=%+v error=%v", conn, err)
	}
}
