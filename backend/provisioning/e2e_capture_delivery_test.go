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

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	memoryauth "github.com/ambi/idmagic/backend/authentication/password/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/adapters/persistence/memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/provisioning/adapters/identitysource"
	memoryprov "github.com/ambi/idmagic/backend/provisioning/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/scim"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
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
	return f.requests[len(f.requests)-1]
}

func (f *fakeSCIMDownstream) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
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
	connRepo := memoryprov.NewProvisioningConnectionRepository()
	deliveryRepo := memoryprov.NewProvisioningDeliveryRepository()
	linkRepo := memoryprov.NewRemoteResourceLinkRepository()

	captureDeps := usecases.CaptureDeps{ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo}
	notifier := usecases.UserMutationNotifier{CaptureDeps: captureDeps}

	adminUserDeps := userusecases.AdminUserDeps{
		UserRepo:             userRepo,
		PasswordHasher:       crypto.NewArgon2idPasswordHasher(),
		PasswordHistoryRepo:  memoryauth.NewPasswordHistoryRepository(),
		ProvisioningNotifier: notifier,
	}

	deliverDeps := usecases.DeliverDeps{
		ConnectionRepo:  connRepo,
		DeliveryRepo:    deliveryRepo,
		LinkRepo:        linkRepo,
		AttributeSource: &identitysource.UserAttributeSource{UserRepo: userRepo},
		NewTargetClient: func(_ *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error) {
			return &scim.Client{HTTPClient: server.Client(), BaseURL: server.URL, BearerToken: secret}, nil
		},
	}

	h := &e2eHarness{
		t: t, userRepo: userRepo, connRepo: connRepo, deliveryRepo: deliveryRepo, linkRepo: linkRepo,
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
		AttributeSource: &identitysource.UserAttributeSource{UserRepo: userRepo},
		NewTargetClient: func(_ *domain.ProvisioningConnection, secret string) (ports.ProvisioningTargetClient, error) {
			return &scim.Client{HTTPClient: server.Client(), BaseURL: server.URL, BearerToken: secret}, nil
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
