// wi-45 T008: end-to-end verification that a real IdManagement mutation
// reaches a real downstream over HTTP through the actual capture → deliver
// wiring (userusecases.CreateUser/UpdateUser/SetUserDisabled/DeleteUser →
// provisioning.UserMutationNotifier → CaptureLifecycleEvent →
// ExecuteDelivery → scim.Client → fake SCIM downstream). This intentionally
// bypasses the Jobs queue/dispatcher (already covered end-to-end by
// backend/jobs/usecases's own runner tests and by
// provisioning/usecases/job_handler_test.go's single-attempt behavior) so the
// test isolates the two seams that had no coverage at all before T008: the
// IdManagement→Provisioning notifier wiring, and ExecuteDelivery driving a
// real scim.Client/UserAttributeSource pair against real HTTP.
package provisioning_test

// 主要ユースケース追跡: REQ-PLATFORM-003、REQ-PROVISIONING-003。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	memoryauth "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	scim "github.com/ambi/idmagic/backend/provisioning/client_scim"
	memoryprov "github.com/ambi/idmagic/backend/provisioning/db_memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	identitysource "github.com/ambi/idmagic/backend/provisioning/source_idmanagement"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// fakeSCIMDownstream records every request it receives and issues sequential
// remote IDs, standing in for a real SCIM 2.0 service provider.
type fakeSCIMDownstream struct {
	mu       sync.Mutex
	requests []recordedRequest
	nextID   int
}

type recordedRequest struct {
	method string
	path   string
	auth   string
	body   map[string]any
}

func (f *fakeSCIMDownstream) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		f.mu.Lock()
		f.requests = append(f.requests, recordedRequest{method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"), body: body})
		f.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/Users":
			f.mu.Lock()
			f.nextID++
			id := fmt.Sprintf("remote-user-%d", f.nextID)
			f.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
		case r.Method == http.MethodPost && r.URL.Path == "/Groups":
			f.mu.Lock()
			f.nextID++
			id := fmt.Sprintf("remote-group-%d", f.nextID)
			f.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
		case (r.Method == http.MethodPut || r.Method == http.MethodPatch):
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (f *fakeSCIMDownstream) last() recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		// 下流へ一度も届かなかった場合は panic ではなく空の記録を返し、
		// 呼び出し側の「何が届くべきか」の表明で失敗させる。
		return recordedRequest{}
	}
	return f.requests[len(f.requests)-1]
}

func (f *fakeSCIMDownstream) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeSCIMDownstream) snapshot() []recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]recordedRequest(nil), f.requests...)
}

// find returns the first recorded request matching method and path, or nil.
func (f *fakeSCIMDownstream) find(method, path string) *recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.requests {
		if f.requests[i].method == method && f.requests[i].path == path {
			return &f.requests[i]
		}
	}
	return nil
}

// e2eHarness wires real memory persistence + real usecases across
// IdManagement and Provisioning, exactly as bootstrap/memory.go does in
// production (minus the HTTP layer and Jobs queue).
type e2eHarness struct {
	t             *testing.T
	userRepo      *usermemory.UserRepository
	connRepo      *memoryprov.ProvisioningConnectionRepository
	deliveryRepo  *memoryprov.ProvisioningDeliveryRepository
	linkRepo      *memoryprov.RemoteResourceLinkRepository
	groupRepo     *groupmemory.GroupRepository
	groupNotifier usecases.GroupMutationNotifier
	adminUserDeps userusecases.AdminUserDeps
	deliverDeps   usecases.DeliverDeps
	downstream    *fakeSCIMDownstream
	server        *httptest.Server
	tenantID      string
	connectionID  string
}

const e2eSecret = "test-bearer-token"

func newE2EHarness(t *testing.T) *e2eHarness {
	t.Helper()
	downstream := &fakeSCIMDownstream{}
	server := httptest.NewServer(downstream.handler())
	t.Cleanup(server.Close)

	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	connRepo := memoryprov.NewProvisioningConnectionRepository()
	deliveryRepo := memoryprov.NewProvisioningDeliveryRepository()
	linkRepo := memoryprov.NewRemoteResourceLinkRepository()

	captureDeps := usecases.CaptureDeps{ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo}
	notifier := usecases.UserMutationNotifier{CaptureDeps: captureDeps}
	groupNotifier := usecases.GroupMutationNotifier{CaptureDeps: captureDeps}

	adminUserDeps := userusecases.AdminUserDeps{
		UserRepo:             userRepo,
		PasswordHasher:       passwords_argon2id.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo:  memoryauth.NewPasswordHistoryRepository(),
		ProvisioningNotifier: notifier,
	}

	deliverDeps := usecases.DeliverDeps{
		ConnectionRepo: connRepo,
		DeliveryRepo:   deliveryRepo,
		LinkRepo:       linkRepo,
		AttributeSource: identitysource.CombinedAttributeSource{
			User:  &identitysource.UserAttributeSource{UserRepo: userRepo},
			Group: &identitysource.GroupAttributeSource{GroupRepo: groupRepo, UserRepo: userRepo},
		},
		NewTargetClient: func(_ *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error) {
			return scim.NewBearerTokenClient(server.Client(), server.URL, secret), nil
		},
	}

	h := &e2eHarness{
		t: t, userRepo: userRepo, connRepo: connRepo, deliveryRepo: deliveryRepo, linkRepo: linkRepo,
		groupRepo: groupRepo, groupNotifier: groupNotifier,
		adminUserDeps: adminUserDeps, deliverDeps: deliverDeps, downstream: downstream, server: server,
		tenantID: tenancydomain.DefaultTenantID, connectionID: "app-e2e",
	}
	h.registerActiveConnection()
	return h
}

func (h *e2eHarness) registerActiveConnection() {
	h.t.Helper()
	now := time.Now().UTC()
	conn := &domain.ProvisioningConnection{
		ApplicationID: h.connectionID, TenantID: h.tenantID, Status: domain.ConnectionActive,
		BaseURL:    h.server.URL,
		Credential: domain.ProvisioningConnectionCredentialMetadata{CredentialID: "cred-e2e", AuthMethod: domain.AuthBearerToken, CreatedAt: now},
		FeatureFlags: domain.ProvisioningFeatureFlags{
			CreateUsers: true, UpdateUsers: true, DeactivateUsers: true, DeleteUsers: true,
		},
		Scope:    domain.ScopeAllUsers,
		Matching: domain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy: domain.DeprovisionPolicy{
			OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate,
		},
		RateLimitPerMinute: 60, MaxAttempts: 8, QuarantineAfterConsecutiveFailure: 10,
		Health: domain.HealthOK, CreatedAt: now, UpdatedAt: now,
	}
	if err := h.connRepo.Register(context.Background(), conn, e2eSecret); err != nil {
		h.t.Fatalf("Register() error = %v", err)
	}
}

// pendingDeliveryFor finds the single pending delivery CaptureLifecycleEvent
// created for userID and executes it, returning the resulting delivery.
func (h *e2eHarness) executePendingDelivery(userID string) *domain.ProvisioningDelivery {
	h.t.Helper()
	ctx := context.Background()
	deliveries, err := h.deliveryRepo.ListByConnection(ctx, h.tenantID, h.connectionID, nil, 10)
	if err != nil {
		h.t.Fatalf("ListByConnection() error = %v", err)
	}
	var target *domain.ProvisioningDelivery
	for _, d := range deliveries {
		if d.SourceID == userID && d.Status == domain.DeliveryPending {
			target = d
		}
	}
	if target == nil {
		h.t.Fatalf("no pending delivery found for user %s", userID)
	}
	if err := usecases.ExecuteDelivery(ctx, h.deliverDeps, h.tenantID, target.ID, time.Now().UTC()); err != nil {
		h.t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	got, err := h.deliveryRepo.Find(ctx, h.tenantID, target.ID)
	if err != nil {
		h.t.Fatalf("Find() error = %v", err)
	}
	return got
}

func TestE2E_CreateUpdateDisableDelete_ReachesRealDownstream(t *testing.T) {
	h := newE2EHarness(t)
	ctx := context.Background()

	// 1. Create: IdManagement.CreateUser -> real ProvisioningNotifier -> real
	// CaptureLifecycleEvent -> real ExecuteDelivery -> real scim.Client POST.
	user, err := userusecases.CreateUser(ctx, h.adminUserDeps, userusecases.CreateUserInput{
		PreferredUsername: "alice-e2e", Password: "correct-horse-battery-staple-9", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	created := h.executePendingDelivery(user.ID)
	if created.Status != domain.DeliverySucceeded {
		t.Fatalf("create delivery status = %v, want succeeded (last_error=%v)", created.Status, created.LastError)
	}
	if got := h.downstream.last(); got.method != http.MethodPost || got.path != "/Users" {
		t.Errorf("downstream last request = %+v, want POST /Users", got)
	}
	if h.downstream.last().auth != "Bearer "+e2eSecret {
		t.Errorf("downstream Authorization = %q, want Bearer %s", h.downstream.last().auth, e2eSecret)
	}
	link, err := h.linkRepo.Find(ctx, h.connectionID, domain.SourceTypeUser, user.ID)
	if err != nil || link == nil || link.RemoteID != "remote-user-1" {
		t.Fatalf("RemoteResourceLink after create = %+v, err=%v, want remote_id=remote-user-1", link, err)
	}

	// 2. Update: UpdateUser -> ProvisioningUserAttributesChanged -> update delivery -> PUT/PATCH.
	newName := "Alice E2E"
	if _, err := userusecases.UpdateUser(ctx, h.adminUserDeps, userusecases.UpdateUserInput{
		Sub: user.ID, Name: &newName, Now: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("UpdateUser() error = %v", err)
	}
	updated := h.executePendingDelivery(user.ID)
	if updated.Status != domain.DeliverySucceeded {
		t.Fatalf("update delivery status = %v, want succeeded (last_error=%v)", updated.Status, updated.LastError)
	}
	if got := h.downstream.last(); got.method != http.MethodPut && got.method != http.MethodPatch {
		t.Errorf("downstream last request method = %q, want PUT or PATCH", got.method)
	}

	// 3. Disable: SetUserDisabled(true) -> deactivate delivery.
	if _, err := userusecases.SetUserDisabled(ctx, h.adminUserDeps, "actor", user.ID, true, time.Now().UTC()); err != nil {
		t.Fatalf("SetUserDisabled() error = %v", err)
	}
	disabled := h.executePendingDelivery(user.ID)
	if disabled.Status != domain.DeliverySucceeded || disabled.Operation != domain.OperationDeactivate {
		t.Fatalf("disable delivery = %+v, want succeeded deactivate", disabled)
	}

	// 4. Delete: DeleteUser -> deprovision per policy (on_delete=deactivate here) -> delivery.
	if err := userusecases.DeleteUser(ctx, h.adminUserDeps, userusecases.DeleteUserInput{
		ActorUserID: "actor", Sub: user.ID, Now: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	deleted := h.executePendingDelivery(user.ID)
	if deleted.Status != domain.DeliverySucceeded || deleted.Operation != domain.OperationDeactivate {
		t.Fatalf("delete delivery = %+v, want succeeded deactivate (on_delete policy)", deleted)
	}

	if got := h.downstream.count(); got != 4 {
		t.Errorf("total downstream requests = %d, want 4 (create, update, disable, delete)", got)
	}
}

// TestE2E_DeleteWithDeleteOnPolicy_SendsRealDELETE verifies that
// DeprovisionPolicy.OnDelete=delete drives a real DELETE against the
// downstream (distinct from the default deactivate-on-delete scenario above).
func TestE2E_DeleteWithDeleteOnPolicy_SendsRealDELETE(t *testing.T) {
	h := newE2EHarness(t)
	ctx := context.Background()
	conn, err := h.connRepo.Find(ctx, h.tenantID, h.connectionID)
	if err != nil || conn == nil {
		t.Fatalf("Find() connection error = %v", err)
	}
	conn.DeprovisionPolicy.OnDelete = domain.DeprovisionDelete
	if err := h.connRepo.Update(ctx, conn, nil); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	user, err := userusecases.CreateUser(ctx, h.adminUserDeps, userusecases.CreateUserInput{
		PreferredUsername: "bob-e2e", Password: "correct-horse-battery-staple-9", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	h.executePendingDelivery(user.ID)

	if err := userusecases.DeleteUser(ctx, h.adminUserDeps, userusecases.DeleteUserInput{
		ActorUserID: "actor", Sub: user.ID, Now: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	deleted := h.executePendingDelivery(user.ID)
	if deleted.Status != domain.DeliverySucceeded || deleted.Operation != domain.OperationDelete {
		t.Fatalf("delete delivery = %+v, want succeeded delete", deleted)
	}
	if got := h.downstream.last(); got.method != http.MethodDelete {
		t.Errorf("downstream last request method = %q, want DELETE", got.method)
	}
}

// TestE2E_TransientFailureThenSuccess_ConvergesAcrossRetries drives
// ExecuteDelivery repeatedly against a downstream that fails twice (503) then
// succeeds, mirroring how the Jobs runner retries a non-terminal failure
// (backend/jobs/usecases.Runner, tested generically elsewhere) but proving the
// provisioning-specific path actually converges end-to-end.
func TestE2E_TransientFailureThenSuccess_ConvergesAcrossRetries(t *testing.T) {
	var mu sync.Mutex
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		attempts++
		n := attempts
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": "remote-after-retries"})
	}))
	t.Cleanup(server.Close)

	userRepo := usermemory.NewUserRepository()
	groupRepo := groupmemory.NewGroupRepository()
	connRepo := memoryprov.NewProvisioningConnectionRepository()
	deliveryRepo := memoryprov.NewProvisioningDeliveryRepository()
	linkRepo := memoryprov.NewRemoteResourceLinkRepository()
	ctx := context.Background()
	now := time.Now().UTC()
	conn := &domain.ProvisioningConnection{
		ApplicationID: "app-retry", TenantID: tenancydomain.DefaultTenantID, Status: domain.ConnectionActive, BaseURL: server.URL,
		Credential:         domain.ProvisioningConnectionCredentialMetadata{CredentialID: "cred-retry", AuthMethod: domain.AuthBearerToken, CreatedAt: now},
		FeatureFlags:       domain.ProvisioningFeatureFlags{CreateUsers: true},
		Scope:              domain.ScopeAllUsers,
		Matching:           domain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy:  domain.DeprovisionPolicy{OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate},
		RateLimitPerMinute: 60, MaxAttempts: 8, QuarantineAfterConsecutiveFailure: 10, Health: domain.HealthOK,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := connRepo.Register(ctx, conn, "secret"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := userRepo.Save(ctx, &userdomain.User{
		ID: "user-retry", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "carol-e2e",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("Save() user error = %v", err)
	}
	deps := usecases.CaptureDeps{ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo}
	if err := usecases.CaptureLifecycleEvent(ctx, deps, tenancydomain.DefaultTenantID, domain.SourceTypeUser, "user-retry", ports.TriggerUserCreated, "", now); err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, err := deliveryRepo.ListByConnection(ctx, tenancydomain.DefaultTenantID, "app-retry", nil, 10)
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("ListByConnection() = %v, err=%v, want 1 delivery", deliveries, err)
	}
	deliverDeps := usecases.DeliverDeps{
		ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo, LinkRepo: linkRepo,
		AttributeSource: identitysource.CombinedAttributeSource{
			User:  &identitysource.UserAttributeSource{UserRepo: userRepo},
			Group: &identitysource.GroupAttributeSource{GroupRepo: groupRepo, UserRepo: userRepo},
		},
		NewTargetClient: func(_ *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error) {
			return scim.NewBearerTokenClient(server.Client(), server.URL, secret), nil
		},
	}

	// Attempt 1 and 2 fail (503, non-terminal from the caller's perspective —
	// the Jobs runner would retry); attempt 3 succeeds.
	for i := range 2 {
		if err := usecases.ExecuteDelivery(ctx, deliverDeps, tenancydomain.DefaultTenantID, deliveries[0].ID, time.Now().UTC()); err == nil {
			t.Fatalf("ExecuteDelivery() attempt %d: want error (downstream returns 503), got nil", i+1)
		}
	}
	if err := usecases.ExecuteDelivery(ctx, deliverDeps, tenancydomain.DefaultTenantID, deliveries[0].ID, time.Now().UTC()); err != nil {
		t.Fatalf("ExecuteDelivery() attempt 3: want success after transient failures, got %v", err)
	}
	got, err := deliveryRepo.Find(ctx, tenancydomain.DefaultTenantID, deliveries[0].ID)
	if err != nil || got.Status != domain.DeliverySucceeded {
		t.Fatalf("final delivery status = %+v, err=%v, want succeeded", got, err)
	}
	if attempts != 3 {
		t.Errorf("downstream attempts = %d, want 3", attempts)
	}
}

// wi-441: `push_groups` を有効にした接続で、Group の変更が下流への書き込みまで
// 届くこと。正本文書は Push Groups を能力として宣言しているのに、捕捉から配信
// までの経路がどこにも配線されておらず、設定は保存され画面は有効と表示しながら
// 配信は 1 件も生まれていなかった。失敗として現れないぶん、気付く手掛かりが無い。
func TestE2E_GroupChange_ReachesRealDownstream(t *testing.T) {
	h := newE2EHarness(t)
	h.enablePushGroups()
	ctx := context.Background()
	now := time.Now().UTC()

	group := h.seedGroup()

	// 変更を捕捉させる。IdManagement の Group 側から呼ばれる通知先である。
	if err := h.groupNotifier.NotifyGroupMutation(
		ctx, h.tenantID, group.ID, groupports.ProvisioningGroupCreated, now,
	); err != nil {
		t.Fatalf("NotifyGroupMutation() error = %v", err)
	}

	delivery := h.executePendingGroupDelivery(group.ID)
	if delivery.Status != domain.DeliverySucceeded {
		t.Fatalf("delivery status = %q, want succeeded (last_error=%v)", delivery.Status, delivery.LastError)
	}

	created := h.downstream.find(http.MethodPost, "/Groups")
	if created == nil {
		t.Fatalf("下流へ POST /Groups が届いていない: %+v", h.downstream.snapshot())
	}
	if created.body["displayName"] != "engineering" {
		t.Fatalf("POST /Groups body = %+v, want displayName=engineering", created.body)
	}
	// 相関が残っていること。次の変更が作成ではなく更新になる根拠である。
	link, err := h.linkRepo.Find(ctx, h.connectionID, domain.SourceTypeGroup, group.ID)
	if err != nil || link == nil || link.RemoteID == "" {
		t.Fatalf("RemoteResourceLink = (%+v, %v), want a link carrying the downstream id", link, err)
	}
}

// push_groups が無効なら Group の配信は 1 件も生まれない。能力を実装したことで
// 逆に、無効にしている接続へ書き込みが始まっていないことを固定する。
func TestE2E_GroupChange_ProducesNothingWhenPushGroupsIsOff(t *testing.T) {
	h := newE2EHarness(t) // push_groups は既定で false
	ctx := context.Background()
	group := h.seedGroup()

	if err := h.groupNotifier.NotifyGroupMutation(
		ctx, h.tenantID, group.ID, groupports.ProvisioningGroupCreated, time.Now().UTC(),
	); err != nil {
		t.Fatalf("NotifyGroupMutation() error = %v", err)
	}

	deliveries, err := h.deliveryRepo.ListByConnection(ctx, h.tenantID, h.connectionID, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range deliveries {
		if d.SourceType == domain.SourceTypeGroup {
			t.Fatalf("push_groups が無効なのに Group の配信が生まれている: %+v", d)
		}
	}
	// 下流へも何も届いていない。配信が無いことと、書き込みが無いことは別の主張である。
	if got := h.downstream.snapshot(); len(got) != 0 {
		t.Fatalf("下流への要求 = %+v, want none", got)
	}
}

// enablePushGroups は登録済みの接続で push_groups を有効にする。
// 対象の選び方は既定の assigned_groups とし、表示名は Group の name から採る。
func (h *e2eHarness) enablePushGroups() {
	h.t.Helper()
	ctx := context.Background()
	conn, err := h.connRepo.Find(ctx, h.tenantID, h.connectionID)
	if err != nil || conn == nil {
		h.t.Fatalf("Find() = (%+v, %v)", conn, err)
	}
	conn.FeatureFlags.PushGroups = true
	conn.GroupPush = &domain.GroupPushConfig{
		Selection: domain.GroupSelectionAssignedGroups, DisplayNameSource: "name",
	}
	conn.AttributeMappings = append(conn.AttributeMappings,
		domain.AttributeMappingRule{
			TargetPath: "displayName", SourceKind: domain.SourceKindAttribute,
			SourceKey: "display_name", ApplyOn: domain.ApplyCreateAndUpdate,
		})
	if err := h.connRepo.Update(ctx, conn, nil); err != nil {
		h.t.Fatalf("Update() error = %v", err)
	}
}

// seedGroup は 1 つの Group を置く。id と表示名は固定で、対象の選択を見る
// 検査が明示指定に書く id (group-eng) と揃えてある。
func (h *e2eHarness) seedGroup() *groupdomain.Group {
	h.t.Helper()
	const id, name = "group-eng", "engineering"
	now := time.Now().UTC()
	group := &groupdomain.Group{
		ID: id, TenantID: h.tenantID, Name: name,
		MembershipType: groupdomain.GroupMembershipManual,
		CreatedAt:      now, UpdatedAt: now,
	}
	if err := h.groupRepo.Save(context.Background(), group); err != nil {
		h.t.Fatalf("Save() error = %v", err)
	}
	return group
}

// executePendingDeliveryFor は sourceType/sourceID に対する pending の配信を 1 件実行する。
// executePendingGroupDelivery は sourceID に対する pending の Group 配信を 1 件実行する。
func (h *e2eHarness) executePendingGroupDelivery(sourceID string) *domain.ProvisioningDelivery {
	h.t.Helper()
	const sourceType = domain.SourceTypeGroup
	ctx := context.Background()
	deliveries, err := h.deliveryRepo.ListByConnection(ctx, h.tenantID, h.connectionID, nil, 20)
	if err != nil {
		h.t.Fatalf("ListByConnection() error = %v", err)
	}
	var target *domain.ProvisioningDelivery
	for _, d := range deliveries {
		if d.SourceType == sourceType && d.SourceID == sourceID && d.Status == domain.DeliveryPending {
			target = d
		}
	}
	if target == nil {
		h.t.Fatalf("no pending %s delivery for %s (deliveries=%d)", sourceType, sourceID, len(deliveries))
	}
	if err := usecases.ExecuteDelivery(ctx, h.deliverDeps, h.tenantID, target.ID, time.Now().UTC()); err != nil {
		h.t.Fatalf("ExecuteDelivery() error = %v", err)
	}
	got, err := h.deliveryRepo.Find(ctx, h.tenantID, target.ID)
	if err != nil {
		h.t.Fatalf("Find() error = %v", err)
	}
	return got
}

// メンバーシップの変更が下流への members PATCH まで届くこと。
// 送るのは既に下流へ provision 済みのメンバーだけである。相関の無い User を
// 送ると、下流はこの接続が持たないリソースを作りかねない。
func TestE2E_GroupMembership_PatchesOnlyProvisionedMembers(t *testing.T) {
	h := newE2EHarness(t)
	h.enablePushGroups()
	h.deliverDeps.GroupMemberSource = &identitysource.GroupMemberSource{GroupRepo: h.groupRepo}
	ctx := context.Background()
	now := time.Now().UTC()

	group := h.seedGroup()
	// 1 人は下流へ provision 済み、もう 1 人はまだである。
	provisioned, err := userusecases.CreateUser(ctx, h.adminUserDeps, userusecases.CreateUserInput{
		PreferredUsername: "alice-member", Password: "correct-horse-battery-staple-9", Now: now,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	h.executePendingDelivery(provisioned.ID)
	h.addMember(group.ID, provisioned.ID)
	h.addMember(group.ID, "user-never-provisioned")

	if err := h.groupNotifier.NotifyGroupMutation(
		ctx, h.tenantID, group.ID, groupports.ProvisioningGroupMembershipChanged, now,
	); err != nil {
		t.Fatalf("NotifyGroupMutation() error = %v", err)
	}
	delivery := h.executePendingGroupDelivery(group.ID)
	if delivery.Status != domain.DeliverySucceeded {
		t.Fatalf("delivery status = %q, want succeeded (last_error=%v)", delivery.Status, delivery.LastError)
	}

	patch := h.downstream.findGroupMemberPatch()
	if patch == nil {
		t.Fatalf("members の PATCH が届いていない: %+v", h.downstream.snapshot())
	}
	op, members := membersOfPatch(t, patch)
	// 増分の `add` であること。`remove` を送れば、下流のメンバーを消してしまう。
	if op != "add" {
		t.Fatalf("PATCH op = %q, want add", op)
	}
	if len(members) != 1 {
		t.Fatalf("PATCH members = %v, want exactly the provisioned member", members)
	}
	// 相関を持たないメンバーの id をこちらで作って送っていないこと。
	if members[0] == "user-never-provisioned" {
		t.Fatalf("下流の相関が無いメンバーを送っている: %v", members)
	}
}

// findGroupMemberPatch は Group への members PATCH を返す。
func (f *fakeSCIMDownstream) findGroupMemberPatch() *recordedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.requests {
		r := f.requests[i]
		if r.method != http.MethodPatch {
			continue
		}
		ops, _ := r.body["Operations"].([]any)
		for _, op := range ops {
			if m, ok := op.(map[string]any); ok && m["path"] == "members" {
				return &f.requests[i]
			}
		}
	}
	return nil
}

// membersOfPatch returns the members Operation's op and the member ids it names.
func membersOfPatch(t *testing.T, request *recordedRequest) (string, []string) {
	t.Helper()
	ops, _ := request.body["Operations"].([]any)
	for _, op := range ops {
		m, ok := op.(map[string]any)
		if !ok || m["path"] != "members" {
			continue
		}
		opName, _ := m["op"].(string)
		values, _ := m["value"].([]any)
		out := make([]string, 0, len(values))
		for _, v := range values {
			if entry, ok := v.(map[string]any); ok {
				if id, ok := entry["value"].(string); ok {
					out = append(out, id)
				}
			}
		}
		return opName, out
	}
	t.Fatalf("members の Operation が見つからない: %+v", request.body)
	return "", nil
}

func (h *e2eHarness) addMember(groupID, userID string) {
	h.t.Helper()
	if _, err := h.groupRepo.AddMember(context.Background(), &groupdomain.GroupMember{
		GroupID: groupID, UserID: userID,
		Source: groupdomain.MembershipSourceManual, CreatedAt: time.Now().UTC(),
	}); err != nil {
		h.t.Fatalf("AddMember() error = %v", err)
	}
}

// 2 つの番人を分けて固定する。`push_groups` の機能フラグと、GroupPushConfig に
// よる対象の選択は別々に配信を止めるので、片方だけを外した誤実装がもう片方に
// 隠れないよう、条件を 1 つずつ変えた検査を置く。
func TestE2E_GroupChange_EachGuardStopsDeliveryOnItsOwn(t *testing.T) {
	ctx := context.Background()

	t.Run("対象の設定はあるが push_groups が無効なら配信しない", func(t *testing.T) {
		h := newE2EHarness(t)
		h.enablePushGroups()
		h.setPushGroups(false) // 対象の設定は残したまま、機能フラグだけを落とす。
		group := h.seedGroup()

		h.notifyGroupCreated(group.ID)
		h.assertNoGroupDelivery()
	})

	t.Run("push_groups は有効だが対象に含まれないなら配信しない", func(t *testing.T) {
		h := newE2EHarness(t)
		h.enablePushGroups()
		// 明示指定で、別の Group だけを対象にする。機能フラグは有効のままである。
		h.setGroupPush(&domain.GroupPushConfig{
			Selection: domain.GroupSelectionExplicit, ExplicitGroupIDs: []string{"group-other"},
		})
		group := h.seedGroup()

		h.notifyGroupCreated(group.ID)
		h.assertNoGroupDelivery()
	})

	t.Run("push_groups は有効だが対象の設定が無いなら配信しない", func(t *testing.T) {
		// fail-closed: 能力を有効にしただけで対象を決めていない接続へ、
		// 全 Group を送り始めない。GroupPushConfig が無いことは
		// 「まだ設定されていない」であって「すべて」ではない。
		h := newE2EHarness(t)
		h.setPushGroups(true)
		h.setGroupPush(nil)
		group := h.seedGroup()

		h.notifyGroupCreated(group.ID)
		h.assertNoGroupDelivery()
	})

	t.Run("明示指定に含まれていれば配信する", func(t *testing.T) {
		// 上の 2 件が「何も配信しない実装」でも通ってしまわないための対である。
		h := newE2EHarness(t)
		h.enablePushGroups()
		h.setGroupPush(&domain.GroupPushConfig{
			Selection: domain.GroupSelectionExplicit, ExplicitGroupIDs: []string{"group-eng"},
		})
		group := h.seedGroup()

		h.notifyGroupCreated(group.ID)
		delivery := h.executePendingGroupDelivery(group.ID)
		if delivery.Status != domain.DeliverySucceeded {
			t.Fatalf("delivery status = %q, want succeeded", delivery.Status)
		}
		if h.downstream.find(http.MethodPost, "/Groups") == nil {
			t.Fatalf("下流へ POST /Groups が届いていない: %+v", h.downstream.snapshot())
		}
	})
	_ = ctx
}

func (h *e2eHarness) notifyGroupCreated(groupID string) {
	h.t.Helper()
	if err := h.groupNotifier.NotifyGroupMutation(
		context.Background(), h.tenantID, groupID, groupports.ProvisioningGroupCreated, time.Now().UTC(),
	); err != nil {
		h.t.Fatalf("NotifyGroupMutation() error = %v", err)
	}
}

func (h *e2eHarness) assertNoGroupDelivery() {
	h.t.Helper()
	deliveries, err := h.deliveryRepo.ListByConnection(context.Background(), h.tenantID, h.connectionID, nil, 20)
	if err != nil {
		h.t.Fatal(err)
	}
	for _, d := range deliveries {
		if d.SourceType == domain.SourceTypeGroup {
			h.t.Fatalf("Group の配信が生まれている: %+v", d)
		}
	}
	if got := h.downstream.snapshot(); len(got) != 0 {
		h.t.Fatalf("下流への要求 = %+v, want none", got)
	}
}

func (h *e2eHarness) setPushGroups(enabled bool) {
	h.t.Helper()
	conn := h.connection()
	conn.FeatureFlags.PushGroups = enabled
	h.saveConnection(conn)
}

func (h *e2eHarness) setGroupPush(config *domain.GroupPushConfig) {
	h.t.Helper()
	conn := h.connection()
	conn.GroupPush = config
	h.saveConnection(conn)
}

func (h *e2eHarness) connection() *domain.ProvisioningConnection {
	h.t.Helper()
	conn, err := h.connRepo.Find(context.Background(), h.tenantID, h.connectionID)
	if err != nil || conn == nil {
		h.t.Fatalf("Find() = (%+v, %v)", conn, err)
	}
	return conn
}

func (h *e2eHarness) saveConnection(conn *domain.ProvisioningConnection) {
	h.t.Helper()
	if err := h.connRepo.Update(context.Background(), conn, nil); err != nil {
		h.t.Fatalf("Update() error = %v", err)
	}
}

// Group の削除は下流の DELETE まで届く。User と違い deprovision policy を
// 借りないので、無効化に読み替えられていないことを方法で確かめる。
func TestE2E_GroupDeleted_SendsRealDELETE(t *testing.T) {
	h := newE2EHarness(t)
	h.enablePushGroups()
	group := h.seedGroup()

	// まず作成を届けて相関を作る。相関が無ければ削除は何もせず成功で終わる。
	h.notifyGroupCreated(group.ID)
	if got := h.executePendingGroupDelivery(group.ID); got.Status != domain.DeliverySucceeded {
		t.Fatalf("create delivery status = %q", got.Status)
	}
	link, err := h.linkRepo.Find(context.Background(), h.connectionID, domain.SourceTypeGroup, group.ID)
	if err != nil || link == nil {
		t.Fatalf("RemoteResourceLink = (%+v, %v)", link, err)
	}

	if err := h.groupNotifier.NotifyGroupMutation(
		context.Background(), h.tenantID, group.ID, groupports.ProvisioningGroupDeleted, time.Now().UTC(),
	); err != nil {
		t.Fatalf("NotifyGroupMutation() error = %v", err)
	}
	deleted := h.executePendingGroupDelivery(group.ID)
	if deleted.Status != domain.DeliverySucceeded {
		t.Fatalf("delete delivery status = %q (last_error=%v)", deleted.Status, deleted.LastError)
	}
	if got := h.downstream.find(http.MethodDelete, "/Groups/"+link.RemoteID); got == nil {
		t.Fatalf("DELETE /Groups/%s が届いていない: %+v", link.RemoteID, h.downstream.snapshot())
	}
}
