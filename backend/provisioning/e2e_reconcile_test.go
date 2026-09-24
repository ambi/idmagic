package provisioning_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

// 照合の E2E は、User の変更を IdManagement の実際のユースケースで起こし、書き込み時の捕捉を
// 通さずに、照合 → 実行 → SCIM クライアント → 下流の HTTP までを本物の部品でつなぐ。

// failingNotifier は、書き込み時の捕捉が失敗した状況を表す。
type failingNotifier struct{}

func (failingNotifier) NotifyUserMutation(context.Context, string, string, userports.ProvisioningTrigger, time.Time) error {
	return errors.New("capture unavailable")
}

// mapActive は接続に active の対応付けを加え、下流へ送る本文で有効状態を観測できるようにする。
func (h *e2eHarness) mapActive() {
	h.t.Helper()
	conn := h.connection()
	conn.AttributeMappings = []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "active", SourceKind: domain.SourceKindAttribute, SourceKey: "active", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
	}
	h.saveConnection(conn)
}

func (h *e2eHarness) reconcileDeps() usecases.ReconcileDeps {
	return usecases.ReconcileDeps{
		ConnectionRepo: h.connRepo, TaskRepo: h.taskRepo, LinkRepo: h.linkRepo, UserRepo: h.userRepo,
	}
}

// provisionActiveUser は書き込み時の捕捉で User を作り、下流へ作成を反映する。
func (h *e2eHarness) provisionActiveUser(username string) string {
	h.t.Helper()
	user, err := userusecases.CreateUser(context.Background(), h.adminUserDeps, userusecases.CreateUserInput{
		PreferredUsername: username, Password: "correct-horse-battery-staple-9", Now: time.Now().UTC(),
	})
	if err != nil {
		h.t.Fatalf("CreateUser() error = %v", err)
	}
	if created := h.executePendingTask(user.ID); created.Status != domain.TaskSucceeded {
		h.t.Fatalf("create task status = %v, want succeeded (last_error=%v)", created.Status, created.LastError)
	}
	return user.ID
}

func (h *e2eHarness) assertNoPendingTask(userID string) {
	h.t.Helper()
	tasks, err := h.taskRepo.ListByConnection(context.Background(), h.tenantID, h.connectionID, nil, 100)
	if err != nil {
		h.t.Fatalf("ListByConnection() error = %v", err)
	}
	for _, task := range tasks {
		if task.SourceID == userID && task.Status == domain.TaskPending {
			h.t.Fatalf("pending task already exists before reconciliation: %+v", task)
		}
	}
}

// reconcileAndExecuteDeactivation は照合を 1 回走らせ、作られた無効化を下流へ反映する。
func (h *e2eHarness) reconcileAndExecuteDeactivation(userID string) {
	h.t.Helper()
	remoteID := h.remoteUserID(userID)
	created, err := usecases.ReconcileConnections(context.Background(), h.reconcileDeps(), 100, time.Now().UTC())
	if err != nil {
		h.t.Fatalf("ReconcileConnections() error = %v", err)
	}
	if created != 1 {
		h.t.Fatalf("ReconcileConnections() created = %d, want 1 deactivation", created)
	}
	task := h.executePendingTask(userID)
	if task.Operation != domain.OperationDeactivate || task.Status != domain.TaskSucceeded {
		h.t.Fatalf("reconciled task = %+v, want succeeded deactivate", task)
	}
	last := h.downstream.last()
	if last.method != http.MethodPut && last.method != http.MethodPatch || last.path != "/Users/"+remoteID {
		h.t.Fatalf("downstream last request = %s %s, want PUT or PATCH /Users/%s", last.method, last.path, remoteID)
	}
	if active, ok := last.body["active"].(bool); !ok || active {
		h.t.Fatalf("downstream body active = %v, want false: %+v", last.body["active"], last.body)
	}
}

//spec:covers REQ-PLATFORM-003, EX-PLATFORM-003-03: 書き込み時の捕捉を呼ばない経路で無効化した User は、次の照合で deactivate のプロビジョニングタスクになり、下流へ active=false が届く。
func TestE2E_ReconcileDeactivatesAUserDisabledWithoutCapture(t *testing.T) {
	h := newE2EHarness(t)
	h.mapActive()
	userID := h.provisionActiveUser("reconcile-uncaptured")

	withoutCapture := h.adminUserDeps
	withoutCapture.ProvisioningNotifier = nil
	if _, err := userusecases.SetUserDisabled(context.Background(), withoutCapture, "actor", userID, true, time.Now().UTC()); err != nil {
		t.Fatalf("SetUserDisabled() error = %v", err)
	}
	h.assertNoPendingTask(userID)

	h.reconcileAndExecuteDeactivation(userID)
}

//spec:covers REQ-PLATFORM-003, EX-PLATFORM-003-02: 書き込み時の捕捉が失敗しても User の無効化はコミットされたままで、次の照合が deactivate のプロビジョニングタスクを作り、下流へ反映する。
func TestE2E_ReconcileRecoversAFailedCapture(t *testing.T) {
	h := newE2EHarness(t)
	h.mapActive()
	userID := h.provisionActiveUser("reconcile-failed-capture")

	failingCapture := h.adminUserDeps
	failingCapture.ProvisioningNotifier = failingNotifier{}
	if _, err := userusecases.SetUserDisabled(context.Background(), failingCapture, "actor", userID, true, time.Now().UTC()); err != nil {
		t.Fatalf("SetUserDisabled() error = %v, want the change to commit despite the failed capture", err)
	}
	user, err := h.userRepo.FindBySub(context.Background(), userID)
	if err != nil || user == nil || user.IsActive() {
		t.Fatalf("user after failed capture = %+v, err=%v, want disabled", user, err)
	}
	h.assertNoPendingTask(userID)

	h.reconcileAndExecuteDeactivation(userID)
}

// 照合は、書き込み時の捕捉が反映済みの状態には何も作らない。これがないと、照合が周期ごとに
// 同じ更新を作り続ける実装を見分けられない。
func TestE2E_ReconcileCreatesNothingWhenCaptureAlreadyConverged(t *testing.T) {
	h := newE2EHarness(t)
	h.provisionActiveUser("reconcile-converged")
	before := h.downstream.count()

	created, err := usecases.ReconcileConnections(context.Background(), h.reconcileDeps(), 100, time.Now().UTC())
	if err != nil {
		t.Fatalf("ReconcileConnections() error = %v", err)
	}
	if created != 0 {
		t.Fatalf("ReconcileConnections() created = %d, want 0 when the downstream already matches", created)
	}
	if got := h.downstream.count(); got != before {
		t.Fatalf("downstream requests = %d, want %d", got, before)
	}
}
