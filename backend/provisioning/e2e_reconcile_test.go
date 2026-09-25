package provisioning_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/provisioning"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	notificationtemplate "github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// インクリメンタル同期の E2E は、User の変更を IdManagement の実際のユースケースで起こし、イベント同期を
// 通さずに、インクリメンタル同期 → 実行 → SCIM クライアント → 下流の HTTP までを本物の部品でつなぐ。

// failingNotifier は、イベント同期が失敗した状況を表す。
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

// provisionActiveUser はイベント同期で User を作り、下流へ作成を反映する。
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

// reconcileAndExecuteDeactivation はインクリメンタル同期を 1 回走らせ、作られた無効化を下流へ反映する。
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

//spec:covers REQ-PLATFORM-003, EX-PLATFORM-003-03: イベント同期を呼ばない経路で無効化した User は、次のインクリメンタル同期で deactivate のプロビジョニングタスクになり、下流へ active=false が届く。
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

//spec:covers REQ-PLATFORM-003, EX-PLATFORM-003-02: イベント同期が失敗しても User の無効化はコミットされたままで、次のインクリメンタル同期が deactivate のプロビジョニングタスクを作り、下流へ反映する。
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

// インクリメンタル同期は、イベント同期が反映済みの状態には何も作らない。これがないと、インクリメンタル同期が周期ごとに
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

// guardRun はインクリメンタル同期を worker と同じ Module.ReconcileDeps の組み立てで走らせ、発行ポートへ渡ったイベントと、
// 本物の Notifier が描画して送信境界へ渡したメールを残す。
type guardRun struct {
	h      *e2eHarness
	events []spec.DomainEvent
	mails  recordingEmailSender
}

// recordingEmailSender は送信境界へ渡ったメールを残す。
type recordingEmailSender struct {
	sent []notificationports.EmailMessage
}

func (s *recordingEmailSender) SendEmail(_ context.Context, message notificationports.EmailMessage) bool {
	s.sent = append(s.sent, message)
	return true
}

func (g *guardRun) reconcile() int {
	g.h.t.Helper()
	module := provisioning.Module{ConnectionRepo: g.h.connRepo, RemoteLinkRepo: g.h.linkRepo, TaskRepo: g.h.taskRepo}
	emit := func(event spec.DomainEvent) { g.events = append(g.events, event) }
	notifier := &notificationtemplate.Notifier{Sender: &g.mails, SystemDefaultLocale: "en"}
	created, err := usecases.ReconcileConnections(context.Background(), module.ReconcileDeps(g.h.userRepo, nil, emit, notifier), 100, time.Now().UTC())
	if err != nil {
		g.h.t.Fatalf("ReconcileConnections() error = %v", err)
	}
	return created
}

func (g *guardRun) quarantines() []*domain.ConnectionQuarantined {
	var quarantined []*domain.ConnectionQuarantined
	for _, event := range g.events {
		if q, ok := event.(*domain.ConnectionQuarantined); ok {
			quarantined = append(quarantined, q)
		}
	}
	return quarantined
}

// setAccidentalDeletionGuard は接続の誤削除ガードの閾値を設定する。nil はその閾値を設定しないことを表す。
func (h *e2eHarness) setAccidentalDeletionGuard(count, percent *int) {
	h.t.Helper()
	conn := h.connection()
	conn.DeprovisionPolicy.AccidentalDeletionCountThreshold = count
	conn.DeprovisionPolicy.AccidentalDeletionPercentThreshold = percent
	h.saveConnection(conn)
}

// narrowScopeToAssignments は接続を assigned_only へ変え、割り当ての無い反映済みの User を
// イベント同期を通さずにスコープ外にする。スコープの設定を誤った状況を表す。
func (h *e2eHarness) narrowScopeToAssignments() {
	h.t.Helper()
	conn := h.connection()
	conn.Scope = domain.ScopeAssignedOnly
	h.saveConnection(conn)
}

func (h *e2eHarness) taskCount() int {
	h.t.Helper()
	tasks, err := h.taskRepo.ListByConnection(context.Background(), h.tenantID, h.connectionID, nil, 1000)
	if err != nil {
		h.t.Fatalf("ListByConnection() error = %v", err)
	}
	return len(tasks)
}

func (h *e2eHarness) health() domain.ProvisioningHealth {
	h.t.Helper()
	return h.connection().Health
}

//spec:covers REQ-PROVISIONING-011: 反映済みの 6 人がイベント同期を通らずにスコープ外になり、件数の閾値 5 を超えると、インクリメンタル同期はプロビジョニングタスクを 1 件も作らず、接続を隔離して ConnectionQuarantined を一度発行する。
func TestE2E_ReconcileOverTheGuardQuarantinesWithoutDeprovisioning(t *testing.T) {
	h := newE2EHarness(t)
	for i := range 6 {
		h.provisionActiveUser(fmt.Sprintf("guard-reconcile-%d", i))
	}
	h.setAccidentalDeletionGuard(new(5), nil)
	h.narrowScopeToAssignments()
	tasksBefore, requestsBefore := h.taskCount(), h.downstream.count()
	run := &guardRun{h: h}

	if created := run.reconcile(); created != 0 {
		t.Fatalf("ReconcileConnections() created = %d, want 0 over the guard", created)
	}
	if got := h.taskCount(); got != tasksBefore {
		t.Fatalf("tasks after reconcile = %d, want %d: no deprovision is created over the guard", got, tasksBefore)
	}
	if got := h.downstream.count(); got != requestsBefore {
		t.Fatalf("downstream requests = %d, want %d", got, requestsBefore)
	}
	if got := h.health(); got != domain.HealthQuarantined {
		t.Fatalf("connection health = %q, want quarantined", got)
	}
	quarantined := run.quarantines()
	if len(quarantined) != 1 || quarantined[0].ApplicationID != h.connectionID || quarantined[0].TenantID != h.tenantID {
		t.Fatalf("ConnectionQuarantined = %+v, want exactly one for %s", quarantined, h.connectionID)
	}
	// 隔離した接続を次のインクリメンタル同期は読まないので、同じ差分から二度目の隔離もタスクも生まれない。
	if created := run.reconcile(); created != 0 || len(run.quarantines()) != 1 {
		t.Fatalf("second reconcile created = %d, quarantines = %d; want 0 and still 1", created, len(run.quarantines()))
	}
}

//spec:covers EX-PROVISIONING-011-01: 件数の閾値 5 を超えるとインクリメンタル同期は deprovision を作らずに接続を隔離して ConnectionQuarantined を発行し、notification_email へ隔離の理由を載せたメールを一通送る。解除すると ProvisioningConnectionQuarantineCleared が発行され health が ok に戻る。
func TestE2E_ReconcileOverTheGuardNotifiesAndIsResumed(t *testing.T) {
	h := newE2EHarness(t)
	for i := range 6 {
		h.provisionActiveUser(fmt.Sprintf("guard-notify-%d", i))
	}
	h.setAccidentalDeletionGuard(new(5), nil)
	conn := h.connection()
	conn.NotificationEmail = new("provisioning-alerts@example.test")
	h.saveConnection(conn)
	h.narrowScopeToAssignments()
	tasksBefore := h.taskCount()
	run := &guardRun{h: h}

	if created := run.reconcile(); created != 0 || h.taskCount() != tasksBefore {
		t.Fatalf("ReconcileConnections() created = %d, tasks = %d; want 0 and %d over the guard", created, h.taskCount(), tasksBefore)
	}
	if got := h.health(); got != domain.HealthQuarantined || len(run.quarantines()) != 1 {
		t.Fatalf("health = %q, ConnectionQuarantined = %d; want quarantined and 1", got, len(run.quarantines()))
	}
	if len(run.mails.sent) != 1 {
		t.Fatalf("mails sent = %d, want exactly one quarantine notice: %+v", len(run.mails.sent), run.mails.sent)
	}
	mail := run.mails.sent[0]
	if mail.To != "provisioning-alerts@example.test" {
		t.Fatalf("mail To = %q, want the connection's notification_email", mail.To)
	}
	reason := *h.connection().QuarantineReason
	for _, want := range []string{h.connectionID, reason} {
		if !strings.Contains(mail.Text, want) || !strings.Contains(mail.HTML, want) {
			t.Fatalf("mail body does not carry %q:\ntext=%s\nhtml=%s", want, mail.Text, mail.HTML)
		}
	}
	// 隔離した接続を次のインクリメンタル同期は読まないので、同じ隔離を二度知らせない。
	run.reconcile()
	if len(run.mails.sent) != 1 {
		t.Fatalf("mails sent after the second reconcile = %d, want still 1", len(run.mails.sent))
	}

	var cleared []spec.DomainEvent
	adminDeps := usecases.AdminDeps{ConnectionRepo: h.connRepo, TaskRepo: h.taskRepo, Emit: func(event spec.DomainEvent) { cleared = append(cleared, event) }}
	if _, err := usecases.ResumeConnection(context.Background(), adminDeps, h.tenantID, h.connectionID, time.Now().UTC()); err != nil {
		t.Fatalf("ResumeConnection() error = %v", err)
	}
	if len(cleared) != 1 || cleared[0].EventType() != "ProvisioningConnectionQuarantineCleared" {
		t.Fatalf("events after resume = %+v, want one ProvisioningConnectionQuarantineCleared", cleared)
	}
	if got := h.health(); got != domain.HealthOK {
		t.Fatalf("health after resume = %q, want ok", got)
	}
}

//spec:covers EX-PROVISIONING-011-04: スコープ外になった反映済みの User が件数の閾値 5 と等しい 5 人なら、インクリメンタル同期は 5 人の deactivate を作り、接続は ok のままで ConnectionQuarantined を発行しない。
func TestE2E_ReconcileAtTheGuardStillDeprovisions(t *testing.T) {
	h := newE2EHarness(t)
	for i := range 5 {
		h.provisionActiveUser(fmt.Sprintf("guard-at-%d", i))
	}
	h.setAccidentalDeletionGuard(new(5), nil)
	h.narrowScopeToAssignments()
	run := &guardRun{h: h}

	if created := run.reconcile(); created != 5 {
		t.Fatalf("ReconcileConnections() created = %d, want 5 deactivations at the threshold", created)
	}
	tasks, err := h.taskRepo.ListByConnection(context.Background(), h.tenantID, h.connectionID, nil, 100)
	if err != nil {
		t.Fatalf("ListByConnection() error = %v", err)
	}
	deactivations := 0
	for _, task := range tasks {
		if task.Status == domain.TaskPending && task.Operation == domain.OperationDeactivate {
			deactivations++
		}
	}
	if deactivations != 5 {
		t.Fatalf("pending deactivations = %d, want 5 (tasks %+v)", deactivations, tasks)
	}
	if got := h.health(); got != domain.HealthOK {
		t.Fatalf("connection health = %q, want ok at the threshold", got)
	}
	if got := run.quarantines(); len(got) != 0 {
		t.Fatalf("ConnectionQuarantined = %+v, want none at the threshold", got)
	}
}
