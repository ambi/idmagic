package provisioning_test

// 主要ユースケース追跡: REQ-PROVISIONING-003、REQ-PROVISIONING-005、REQ-PROVISIONING-006、REQ-PROVISIONING-010、REQ-PROVISIONING-018。

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	appports "github.com/ambi/idmagic/backend/application/ports"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/provisioning"
	scim "github.com/ambi/idmagic/backend/provisioning/client_scim"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	identitysource "github.com/ambi/idmagic/backend/provisioning/source_idmanagement"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// lifecycleRun は worker と同じ組み立て（Module のディスパッチャーとジョブハンドラー、
// Jobs の保存先）で配信を進め、発行ポートへ渡ったイベントを残す。Runner のポーリング
// だけは省き、確保したジョブを直接ハンドラーへ渡す。
type lifecycleRun struct {
	h      *e2eHarness
	module provisioning.Module
	jobs   *jobsmemory.JobRepository
	target string
	events []spec.DomainEvent
}

func newLifecycleRun(h *e2eHarness) *lifecycleRun {
	return &lifecycleRun{
		h:      h,
		module: provisioning.Module{ConnectionRepo: h.connRepo, RemoteLinkRepo: h.linkRepo, DeliveryRepo: h.deliveryRepo},
		jobs:   jobsmemory.NewJobRepository(),
		target: h.server.URL,
	}
}

func (r *lifecycleRun) emit(event spec.DomainEvent) { r.events = append(r.events, event) }

func (r *lifecycleRun) types() []string {
	types := make([]string, 0, len(r.events))
	for _, event := range r.events {
		types = append(types, event.EventType())
	}
	return types
}

func (r *lifecycleRun) dispatch() {
	r.h.t.Helper()
	r.dispatchAt(time.Now().UTC())
}

// dispatchAt は worker の周期処理を now の時点として 1 回実行する。
func (r *lifecycleRun) dispatchAt(now time.Time) {
	r.h.t.Helper()
	if _, err := usecases.DispatchPendingDeliveries(context.Background(), r.module.DispatcherDeps(r.jobs, nil, r.emit), 100, now); err != nil {
		r.h.t.Fatalf("DispatchPendingDeliveries() error = %v", err)
	}
}

// drainAt は now の時点で周期処理を実行し、投入されたジョブをすべて処理する。
func (r *lifecycleRun) drainAt(now time.Time) {
	r.h.t.Helper()
	r.dispatchAt(now)
	for {
		jobs, err := r.jobs.ClaimBatch(context.Background(), "worker-e2e", jobsdomain.LaneDefault, 10, time.Minute, time.Now().UTC())
		if err != nil {
			r.h.t.Fatalf("ClaimBatch() error = %v", err)
		}
		if len(jobs) == 0 {
			return
		}
		for _, job := range jobs {
			if err := r.handle(job); err != nil {
				r.h.t.Fatalf("handle(%s) error = %v", job.ID, err)
			}
			if _, err := r.jobs.Complete(context.Background(), job.ID, "worker-e2e", nil, time.Now().UTC()); err != nil {
				r.h.t.Fatalf("Complete(%s) error = %v", job.ID, err)
			}
		}
	}
}

// claim は投入済みのジョブを 1 件確保する。
func (r *lifecycleRun) claim() *jobsdomain.Job {
	r.h.t.Helper()
	jobs, err := r.jobs.ClaimBatch(context.Background(), "worker-e2e", jobsdomain.LaneDefault, 1, time.Minute, time.Now().UTC())
	if err != nil || len(jobs) != 1 {
		r.h.t.Fatalf("ClaimBatch() = %v, %v; want one job", jobs, err)
	}
	return jobs[0]
}

func (r *lifecycleRun) handle(job *jobsdomain.Job) error {
	attrSource := identitysource.CombinedAttributeSource{
		User:  &identitysource.UserAttributeSource{UserRepo: r.h.userRepo},
		Group: &identitysource.GroupAttributeSource{GroupRepo: r.h.groupRepo, UserRepo: r.h.userRepo},
	}
	newClient := func(_ *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error) {
		return scim.NewBearerTokenClient(http.DefaultClient, r.target, secret), nil
	}
	handler := provisioning.Handler(r.module.JobHandlerDeps(attrSource, nil, newClient, r.emit))
	_, err := handler(context.Background(), job)
	return err
}

func (r *lifecycleRun) delivery(id string) *domain.ProvisioningDelivery {
	r.h.t.Helper()
	d, err := r.h.deliveryRepo.Find(context.Background(), r.h.tenantID, id)
	if err != nil || d == nil {
		r.h.t.Fatalf("Find(%s) = %v, %v", id, d, err)
	}
	return d
}

// wire は監査が保存するのと同じワイヤ表現に変換する。
func wire(t *testing.T, event spec.DomainEvent) map[string]any {
	t.Helper()
	encoded, err := spec.MarshalDomainEvent(event)
	if err != nil {
		t.Fatalf("MarshalDomainEvent(%T) error = %v", event, err)
	}
	var fields map[string]any
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	return fields
}

func (h *e2eHarness) createUser(username string) string {
	h.t.Helper()
	user, err := userusecases.CreateUser(context.Background(), h.adminUserDeps, userusecases.CreateUserInput{
		PreferredUsername: username, Password: "correct-horse-battery-staple-9", Now: time.Now().UTC(),
	})
	if err != nil {
		h.t.Fatalf("CreateUser() error = %v", err)
	}
	return user.ID
}

//spec:covers EX-PROVISIONING-003-01: User の作成から worker の組み立てで配信すると、ProvisioningDeliveryStarted で in_flight になり、下流へ POST して UserProvisioned で succeeded になる。
func TestE2E_DispatchAndDeliveryEmitStartedThenProvisioned(t *testing.T) {
	h := newE2EHarness(t)
	run := newLifecycleRun(h)
	userID := h.createUser("alice-events")

	run.dispatch()
	job := run.claim()
	var params struct {
		DeliveryID string `json:"delivery_id"`
	}
	if err := json.Unmarshal(job.Params, &params); err != nil {
		t.Fatal(err)
	}
	if got := run.delivery(params.DeliveryID); got.Status != domain.DeliveryInFlight || got.JobID == nil || *got.JobID != job.ID {
		t.Fatalf("delivery after dispatch = %+v, want in_flight with job %s", got, job.ID)
	}
	if err := run.handle(job); err != nil {
		t.Fatalf("handle() error = %v", err)
	}
	// 同じジョブの再実行（リースの喪失など）は下流へ再送せず、イベントも増やさない。
	if err := run.handle(job); err != nil {
		t.Fatalf("handle() rerun error = %v", err)
	}

	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted", "UserProvisioned"}) {
		t.Fatalf("events = %v, want [ProvisioningDeliveryStarted UserProvisioned]", got)
	}
	if got := run.delivery(params.DeliveryID); got.Status != domain.DeliverySucceeded || h.downstream.count() != 1 {
		t.Fatalf("status = %v, downstream requests = %d; want succeeded after one POST", got.Status, h.downstream.count())
	}
	started, provisioned := wire(t, run.events[0]), wire(t, run.events[1])
	if started["tenantId"] != h.tenantID || started["deliveryId"] != params.DeliveryID || started["jobId"] != job.ID {
		t.Errorf("ProvisioningDeliveryStarted = %v, want tenant %s, delivery %s, job %s", started, h.tenantID, params.DeliveryID, job.ID)
	}
	if provisioned["userId"] != userID || provisioned["remoteId"] != "remote-user-1" || provisioned["connectionId"] != h.connectionID {
		t.Errorf("UserProvisioned = %v, want user %s provisioned as remote-user-1 on %s", provisioned, userID, h.connectionID)
	}
}

//spec:covers EX-PROVISIONING-005-01: 有効なままの User の割り当て解除は、下流へ active=false を PATCH し UserDeprovisioned（action=deactivate）を発行する。
func TestE2E_UnassignmentSendsActiveFalseAndEmitsDeprovisioned(t *testing.T) {
	h := newE2EHarness(t)
	// 登録時の既定マッピングと同じく、active を User の属性から送る。
	conn := h.connection()
	conn.AttributeMappings = []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "active", SourceKind: domain.SourceKindAttribute, SourceKey: "active", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
	}
	h.saveConnection(conn)
	run := newLifecycleRun(h)
	userID := h.createUser("bob-events")
	run.dispatch()
	if err := run.handle(run.claim()); err != nil {
		t.Fatalf("handle() create error = %v", err)
	}
	remoteID := h.remoteUserID(userID)
	run.events = nil

	notifier := run.module.AssignmentNotifier(nil)
	if err := notifier.NotifyAssignmentMutation(context.Background(), h.tenantID, h.connectionID, userID, appports.ProvisioningAssignmentRemoved, time.Now().UTC()); err != nil {
		t.Fatalf("NotifyAssignmentMutation() error = %v", err)
	}
	run.dispatch()
	if err := run.handle(run.claim()); err != nil {
		t.Fatalf("handle() deactivate error = %v", err)
	}

	last := h.downstream.last()
	if last.path != "/Users/"+remoteID || (last.method != http.MethodPatch && last.method != http.MethodPut) || !sendsInactive(last.body) {
		t.Fatalf("downstream last request = %s %s %v, want an update of %s with active=false", last.method, last.path, last.body, remoteID)
	}
	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted", "UserDeprovisioned"}) {
		t.Fatalf("events = %v, want [ProvisioningDeliveryStarted UserDeprovisioned]", got)
	}
	if deprovisioned := wire(t, run.events[1]); deprovisioned["userId"] != userID || deprovisioned["action"] != "deactivate" {
		t.Errorf("UserDeprovisioned = %v, want user %s with action deactivate", deprovisioned, userID)
	}
}

// sendsInactive は PUT の本文か PATCH の操作のどちらかで active=false を送っているかを返す。
func sendsInactive(body map[string]any) bool {
	if body["active"] == false {
		return true
	}
	operations, _ := body["Operations"].([]any)
	for _, raw := range operations {
		op, _ := raw.(map[string]any)
		if op["path"] == "active" && op["value"] == false {
			return true
		}
		if value, ok := op["value"].(map[string]any); ok && value["active"] == false {
			return true
		}
	}
	return false
}

//spec:covers EX-PROVISIONING-010-01: 下流が失敗し続けると、最後の試行で dead_letter になって UserProvisioningFailed を発行し、連続失敗の閾値で ConnectionQuarantined を発行する。
func TestE2E_ExhaustedDeliveryEmitsFailedAndQuarantined(t *testing.T) {
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(failing.Close)
	h := newE2EHarness(t)
	conn := h.connection()
	conn.QuarantineAfterConsecutiveFailure = 1
	h.saveConnection(conn)
	run := newLifecycleRun(h)
	run.target = failing.URL
	h.createUser("carol-events")

	run.dispatch()
	job := run.claim()
	if err := run.handle(job); err == nil {
		t.Fatal("handle() non-terminal attempt: want the downstream error so Jobs retries")
	}
	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted"}) {
		t.Fatalf("events after a non-terminal failure = %v, want only ProvisioningDeliveryStarted", got)
	}
	job.Attempts = job.MaxAttempts
	if err := run.handle(job); err == nil {
		t.Fatal("handle() terminal attempt: want the downstream error")
	}

	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted", "UserProvisioningFailed", "ConnectionQuarantined"}) {
		t.Fatalf("events = %v, want [ProvisioningDeliveryStarted UserProvisioningFailed ConnectionQuarantined]", got)
	}
	if h.connection().Health != domain.HealthQuarantined {
		t.Errorf("connection health = %v, want quarantined", h.connection().Health)
	}
	if quarantined := wire(t, run.events[2]); quarantined["applicationId"] != h.connectionID || quarantined["consecutiveFailures"] != float64(1) {
		t.Errorf("ConnectionQuarantined = %v, want %s after 1 failure", quarantined, h.connectionID)
	}
}

//spec:covers EX-PROVISIONING-018-01: 必須の属性マッピングを解決できない配信は、下流へ送らず、試行が残っていても UserProvisioningFailed を発行して終わる。
func TestE2E_MissingRequiredAttributeFailsClosedWithoutRetry(t *testing.T) {
	h := newE2EHarness(t)
	conn := h.connection()
	conn.AttributeMappings = []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: `emails[type eq "work"].value`, SourceKind: domain.SourceKindAttribute, SourceKey: "email", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
	}
	h.saveConnection(conn)
	run := newLifecycleRun(h)
	h.createUser("dave-without-email")

	run.dispatch()
	job := run.claim()
	if job.Attempts >= job.MaxAttempts {
		t.Fatalf("job attempts = %d of %d, want attempts left", job.Attempts, job.MaxAttempts)
	}
	if err := run.handle(job); err != nil {
		t.Fatalf("handle() error = %v, want nil so Jobs does not retry", err)
	}

	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted", "UserProvisioningFailed"}) {
		t.Fatalf("events = %v, want [ProvisioningDeliveryStarted UserProvisioningFailed]", got)
	}
	if h.downstream.count() != 0 {
		t.Errorf("downstream requests = %d, want 0", h.downstream.count())
	}
	if failed := wire(t, run.events[1]); failed["sourceType"] != "user" || failed["error"] == "" {
		t.Errorf("UserProvisioningFailed = %v, want a user failure with its reason", failed)
	}
	if h.connection().ConsecutiveFailureCount != 0 {
		t.Errorf("ConsecutiveFailureCount = %d, want 0 for a missing attribute", h.connection().ConsecutiveFailureCount)
	}
}

//spec:covers EX-PROVISIONING-004-01: 下流に存在する User を無効化すると、worker が下流へ active=false を送り UserDeprovisioned を発行する。
func TestE2E_DisablingAUserSendsActiveFalseAndEmitsDeprovisioned(t *testing.T) {
	h := newE2EHarness(t)
	conn := h.connection()
	conn.AttributeMappings = []domain.AttributeMappingRule{
		{TargetPath: "userName", SourceKind: domain.SourceKindAttribute, SourceKey: "preferred_username", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
		{TargetPath: "active", SourceKind: domain.SourceKindAttribute, SourceKey: "active", ApplyOn: domain.ApplyCreateAndUpdate, Required: true},
	}
	h.saveConnection(conn)
	run := newLifecycleRun(h)
	userID := h.createUser("erin-events")
	run.dispatch()
	if err := run.handle(run.claim()); err != nil {
		t.Fatalf("handle() create error = %v", err)
	}
	remoteID := h.remoteUserID(userID)
	run.events = nil

	if _, err := userusecases.SetUserDisabled(context.Background(), h.adminUserDeps, "actor", userID, true, time.Now().UTC()); err != nil {
		t.Fatalf("SetUserDisabled() error = %v", err)
	}
	run.dispatch()
	if err := run.handle(run.claim()); err != nil {
		t.Fatalf("handle() deactivate error = %v", err)
	}

	last := h.downstream.last()
	if last.path != "/Users/"+remoteID || !sendsInactive(last.body) {
		t.Fatalf("downstream last request = %s %s %v, want an update of %s with active=false", last.method, last.path, last.body, remoteID)
	}
	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted", "UserDeprovisioned"}) {
		t.Fatalf("events = %v, want [ProvisioningDeliveryStarted UserDeprovisioned]", got)
	}
}

// useDeleteWithGracePeriod は接続を on_delete=delete、grace_period_days=days にする。
func (h *e2eHarness) useDeleteWithGracePeriod(days int) {
	h.t.Helper()
	conn := h.connection()
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	conn.DeprovisionPolicy.GracePeriodDays = days
	h.saveConnection(conn)
}

func (h *e2eHarness) softDeleteUser(userID string, now time.Time) {
	h.t.Helper()
	if err := userusecases.SoftDeleteUser(context.Background(), h.adminUserDeps, userusecases.SoftDeleteUserInput{
		ActorUserID: "actor", Sub: userID, Now: now,
	}); err != nil {
		h.t.Fatalf("SoftDeleteUser() error = %v", err)
	}
}

//spec:covers EX-PROVISIONING-006-01: 猶予期間 7 日の削除は、7 日目の直前まで下流へ DELETE を送らず、経過後の周期処理で delete の配信を作って DELETE を送り、UserDeprovisioned（action=delete）を発行する。
func TestE2E_DeletionWithAGracePeriodSendsDELETEOnlyAfterItElapses(t *testing.T) {
	h := newE2EHarness(t)
	h.useDeleteWithGracePeriod(7)
	run := newLifecycleRun(h)
	userID := h.provisionUser("frank-grace")
	remoteID := h.remoteUserID(userID)
	deletedAt := time.Now().UTC()

	h.softDeleteUser(userID, deletedAt)
	run.drainAt(deletedAt.Add(7*24*time.Hour - time.Second))
	if got := h.downstream.find(http.MethodDelete, "/Users/"+remoteID); got != nil {
		t.Fatalf("downstream received DELETE %s before the grace period elapsed", got.path)
	}
	run.events = nil

	run.drainAt(deletedAt.Add(7 * 24 * time.Hour))
	if got := h.downstream.find(http.MethodDelete, "/Users/"+remoteID); got == nil {
		t.Fatalf("downstream requests = %v, want DELETE /Users/%s once the grace period elapsed", h.downstream.snapshot(), remoteID)
	}
	if got := run.types(); !slices.Equal(got, []string{"ProvisioningDeliveryStarted", "UserDeprovisioned"}) {
		t.Fatalf("events = %v, want [ProvisioningDeliveryStarted UserDeprovisioned]", got)
	}
	if deprovisioned := wire(t, run.events[1]); deprovisioned["userId"] != userID || deprovisioned["action"] != "delete" {
		t.Errorf("UserDeprovisioned = %v, want user %s with action delete", deprovisioned, userID)
	}
}

//spec:covers EX-PROVISIONING-006-02: 猶予期間内に同じ Application へ再び割り当てると、予約していた delete は取り消され、期限の経過後も下流へ DELETE を送らない。
func TestE2E_ReassignmentWithinTheGracePeriodCancelsTheDELETE(t *testing.T) {
	h := newE2EHarness(t)
	h.useDeleteWithGracePeriod(7)
	run := newLifecycleRun(h)
	userID := h.provisionUser("grace-reassigned")
	remoteID := h.remoteUserID(userID)
	deletedAt := time.Now().UTC()

	h.softDeleteUser(userID, deletedAt)
	notifier := run.module.AssignmentNotifier(nil)
	if err := notifier.NotifyAssignmentMutation(context.Background(), h.tenantID, h.connectionID, userID, appports.ProvisioningAssignmentAdded, deletedAt.Add(24*time.Hour)); err != nil {
		t.Fatalf("NotifyAssignmentMutation() error = %v", err)
	}
	run.drainAt(deletedAt.Add(8 * 24 * time.Hour))

	if got := h.downstream.find(http.MethodDelete, "/Users/"+remoteID); got != nil {
		t.Fatalf("downstream received DELETE %s after the reassignment cancelled it", got.path)
	}
	if got := run.types(); slices.Contains(got, "UserDeprovisioned") {
		t.Errorf("events = %v, want no UserDeprovisioned after the reassignment", got)
	}
}
