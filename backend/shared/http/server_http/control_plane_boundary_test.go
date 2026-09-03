package server_http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys"
	datakeysmemory "github.com/ambi/idmagic/backend/datakeys/db_memory"
	datakeysusecases "github.com/ambi/idmagic/backend/datakeys/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

type observedTenantRepository struct {
	*tenancymemory.TenantRepository
	findAllCalls atomic.Int32
}

func (r *observedTenantRepository) FindAll(ctx context.Context) ([]*tenancydomain.Tenant, error) {
	r.findAllCalls.Add(1)
	return r.TenantRepository.FindAll(ctx)
}

type controlPlaneBoundaryServer struct {
	e        *echo.Echo
	tenants  *observedTenantRepository
	jobs     *jobsmemory.JobRepository
	otherJob *jobdomain.Job
}

func newControlPlaneBoundaryServer(t *testing.T, actor *userdomain.User) *controlPlaneBoundaryServer {
	t.Helper()
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	tenantStore := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
	} {
		if err := tenantStore.Save(t.Context(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	tenants := &observedTenantRepository{TenantRepository: tenantStore}

	users := usermemory.NewUserRepository()
	users.Seed(actor)
	resolver := &fixedAuthnResolver{sub: actor.ID}

	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	dataKeyStore := datakeysmemory.NewDataKeyRepository()
	masterKey, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(masterKey)
	if _, err := datakeysusecases.BootstrapTenantDataKey(t.Context(), datakeysusecases.Deps{Repository: dataKeyStore, Crypto: crypto}, "acme", now); err != nil {
		t.Fatal(err)
	}

	jobStore := jobsmemory.NewJobRepository()
	lane, ok := jobdomain.LaneFor(jobdomain.KindNoopEcho)
	if !ok {
		t.Fatal("noop_echo の実行レーンが登録されていない")
	}
	otherJob, _, err := jobStore.Enqueue(t.Context(), jobports.EnqueueInput{
		TenantID: "acme", Kind: jobdomain.KindNoopEcho, Lane: lane,
		Params: json.RawMessage(`{"message":"other tenant"}`), MaxAttempts: 3, RunAt: now, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := jobStore.Enqueue(t.Context(), jobports.EnqueueInput{
		TenantID: tenancydomain.DefaultTenantID, Kind: jobdomain.KindNoopEcho, Lane: lane,
		Params: json.RawMessage(`{"message":"control plane"}`), MaxAttempts: 3, RunAt: now.Add(time.Minute), Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, UserRepo: users, AuthnResolver: resolver,
		SigningKeys: signingkeys.Module{KeyStore: keyStore},
		DataKeys:    datakeys.Module{Repository: dataKeyStore, Crypto: crypto},
		Jobs:        jobs.Module{Repo: jobStore},
	})
	return &controlPlaneBoundaryServer{e: e, tenants: tenants, jobs: jobStore, otherJob: otherJob}
}

func controlPlaneTestUser(id, tenantID string, roles ...string) *userdomain.User {
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	return &userdomain.User{
		ID: id, TenantID: tenantID, PreferredUsername: id, PasswordHash: "unused", Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

func getControlPlaneBoundary(e *echo.Echo, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
	return rec
}

func assertHealthRefusal(t *testing.T, rec *httptest.ResponseRecorder, findAllCalls int32) {
	t.Helper()
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	for _, leaked := range []string{"tenant_id", "provider_reachable", "active_kid", "acme", "default"} {
		if strings.Contains(rec.Body.String(), leaked) {
			t.Errorf("refusal body contains cross-tenant health data %q: %s", leaked, rec.Body.String())
		}
	}
	if findAllCalls != 0 {
		t.Errorf("TenantRepository.FindAll calls = %d, want 0", findAllCalls)
	}
}

// REQ-SIGNINGKEYS-009: 制御面テナント外の system_admin は、署名鍵ヘルスも横断収集も観測できない。
func TestControlPlaneSigningKeyHealthRejectsSystemAdminOutsideControlPlaneTenant(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin"))
	rec := getControlPlaneBoundary(srv.e, "/realms/acme/api/admin/v1/keys/health")
	assertHealthRefusal(t, rec, srv.tenants.findAllCalls.Load())
}

// REQ-DATAKEYS-006: 制御面テナント外の system_admin は、DEK ヘルスも横断収集も観測できない。
func TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin"))
	rec := getControlPlaneBoundary(srv.e, "/realms/acme/api/admin/v1/data-keys/health")
	assertHealthRefusal(t, rec, srv.tenants.findAllCalls.Load())
}

// REQ-JOBS-012、REQ-JOBS-013: 制御面主体は admin ロールなしで一覧、詳細、取消しを横断できる。
func TestControlPlaneJobOversightNeedsNoAdminRole(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	list := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/jobs?all_tenants=true")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), srv.otherJob.ID) {
		t.Fatalf("cross-tenant list omitted job %q: %s", srv.otherJob.ID, list.Body.String())
	}

	detail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/jobs/"+srv.otherJob.ID)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", detail.Code, detail.Body.String())
	}

	account := getControlPlaneBoundary(srv.e, "/realms/default/api/auth/account")
	if account.Code != http.StatusOK {
		t.Fatalf("account status = %d, body = %s", account.Code, account.Body.String())
	}
	var accountBody struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(account.Body.Bytes(), &accountBody); err != nil {
		t.Fatal(err)
	}
	cookies := account.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("CSRF cookie が返らなかった")
	}
	req := httptest.NewRequest(http.MethodPost, "/realms/default/api/admin/v1/jobs/"+srv.otherJob.ID+"/cancel", http.NoBody)
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", accountBody.CSRFToken)
	req.AddCookie(cookies[0])
	cancel := httptest.NewRecorder()
	srv.e.ServeHTTP(cancel, req)
	if cancel.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, body = %s", cancel.Code, cancel.Body.String())
	}
	got, err := srv.jobs.Get(t.Context(), srv.otherJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != jobdomain.StatusCanceled {
		t.Fatalf("cross-tenant job status = %q, want %q", got.Status, jobdomain.StatusCanceled)
	}
}
