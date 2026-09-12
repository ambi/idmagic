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

	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
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
	e              *echo.Echo
	tenants        *observedTenantRepository
	jobs           *jobsmemory.JobRepository
	otherJob       *jobdomain.Job
	otherAuditID   string
	controlAuditID string
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

	// 監査イベントは 2 テナント分置く。横断参照が「全件返っているか」ではなく
	// 「他テナントの記録が現に混じるか」で判定できるようにするためである。
	auditStore := auditmemory.NewAuditEventStore(0)
	for _, rec := range []*auditports.AuditEventRecord{
		{ID: "audit-acme", TenantID: "acme", Type: "UserCreated", OccurredAt: now},
		{ID: "audit-control", TenantID: tenancydomain.DefaultTenantID, Type: "UserCreated", OccurredAt: now.Add(time.Minute)},
	} {
		if err := auditStore.Append(t.Context(), rec); err != nil {
			t.Fatal(err)
		}
	}

	e := echo.New()
	Register(e, Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, UserRepo: users, AuthnResolver: resolver,
		SigningKeys: signingkeys.Module{KeyStore: keyStore},
		DataKeys:    datakeys.Module{Repository: dataKeyStore, Crypto: crypto},
		Jobs:        jobs.Module{Repo: jobStore},
		Audit:       audit.Module{AuditEventRepo: auditStore},
	})
	return &controlPlaneBoundaryServer{
		e: e, tenants: tenants, jobs: jobStore, otherJob: otherJob,
		otherAuditID: "audit-acme", controlAuditID: "audit-control",
	}
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

// browserPost はブラウザー経路の変更操作を、CSRF トークンと Origin を揃えて送る。
func browserPost(t *testing.T, e *echo.Echo, path string) *httptest.ResponseRecorder {
	t.Helper()
	account := getControlPlaneBoundary(e, "/realms/default/api/auth/account")
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
	req := httptest.NewRequest(http.MethodPost, path, http.NoBody)
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", accountBody.CSRFToken)
	req.AddCookie(cookies[0])
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// REQ-JOBS-015 / EX-JOBS-015-01: 制御面主体はシステム経路で admin ロールなしに一覧、詳細、取消しを横断できる。
func TestControlPlaneJobOversightNeedsNoAdminRole(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	list := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/jobs")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), srv.otherJob.ID) {
		t.Fatalf("cross-tenant list omitted job %q: %s", srv.otherJob.ID, list.Body.String())
	}

	detail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/jobs/"+srv.otherJob.ID)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", detail.Code, detail.Body.String())
	}

	cancel := browserPost(t, srv.e, "/realms/default/api/admin/v1/system/jobs/"+srv.otherJob.ID+"/cancel")
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

// EX-JOBS-015-03: ブラウザーからの要求であることを証明できない取消しは、Job を変えずに拒否される。
func TestSystemJobCancelRefusesWithoutBrowserProof(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	req := httptest.NewRequest(http.MethodPost, "/realms/default/api/admin/v1/system/jobs/"+srv.otherJob.ID+"/cancel", http.NoBody)
	rec := httptest.NewRecorder()
	srv.e.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	// 応答ではなく効果を読む。拒否を書いてから取り消すハンドラーは、応答だけを見る
	// テストを同じように通過する。
	got, err := srv.jobs.Get(t.Context(), srv.otherJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == jobdomain.StatusCanceled {
		t.Fatalf("refused cancel still canceled job %q", srv.otherJob.ID)
	}
}

// REQ-AUDIT-007 / EX-AUDIT-007-01: 制御面主体はシステム経路で全テナントの監査イベントを検索、参照、エクスポートできる。
func TestSystemAuditEventsSpanEveryTenantForControlPlaneActor(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	list := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/audit_events")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", list.Code, list.Body.String())
	}
	for _, want := range []string{srv.otherAuditID, srv.controlAuditID} {
		if !strings.Contains(list.Body.String(), want) {
			t.Fatalf("cross-tenant search omitted audit event %q: %s", want, list.Body.String())
		}
	}

	export := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/audit_events/export")
	if export.Code != http.StatusOK {
		t.Fatalf("export status = %d, body = %s", export.Code, export.Body.String())
	}
	if !strings.Contains(export.Body.String(), srv.otherAuditID) {
		t.Fatalf("cross-tenant export omitted audit event %q: %s", srv.otherAuditID, export.Body.String())
	}

	detail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/audit_events/"+srv.otherAuditID)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", detail.Code, detail.Body.String())
	}
}

// EX-AUDIT-007-02、EX-JOBS-015-02、EX-SYSTEM-020-03: 制御面主体でない実行者はシステム経路から
// どのテナントの記録も観測できない。
func TestSystemRoutesRefuseNonControlPlaneActor(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin", "admin"))

	for _, path := range []string{
		"/realms/acme/api/admin/v1/system/audit_events",
		"/realms/acme/api/admin/v1/system/audit_events/export",
		"/realms/acme/api/admin/v1/system/audit_events/" + srv.otherAuditID,
		"/realms/acme/api/admin/v1/system/jobs",
		"/realms/acme/api/admin/v1/system/jobs/" + srv.otherJob.ID,
	} {
		rec := getControlPlaneBoundary(srv.e, path)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s status = %d, want %d; body = %s", path, rec.Code, http.StatusForbidden, rec.Body.String())
		}
		for _, leaked := range []string{srv.otherAuditID, srv.controlAuditID, srv.otherJob.ID} {
			if strings.Contains(rec.Body.String(), leaked) {
				t.Errorf("%s refusal body leaked %q: %s", path, leaked, rec.Body.String())
			}
		}
	}
}

// REQ-JOBS-012、REQ-JOBS-013 のテナント境界をこの経路で確かめる。
// EX-JOBS-012-04、EX-JOBS-013-05、EX-SYSTEM-020-01: テナント管理経路は制御面主体に対しても
// 要求先テナントへ閉じる。横断を求める入力を添えても範囲は変わらない。
func TestTenantAdminApisStayInsideRequestTenant(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t,
		controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin", "admin"))

	list := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/jobs?all_tenants=true")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", list.Code, list.Body.String())
	}
	if strings.Contains(list.Body.String(), srv.otherJob.ID) {
		t.Errorf("tenant listing crossed into another tenant's job %q: %s", srv.otherJob.ID, list.Body.String())
	}

	events := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/audit_events?all_tenants=true")
	if events.Code != http.StatusOK {
		t.Fatalf("audit status = %d, body = %s", events.Code, events.Body.String())
	}
	if strings.Contains(events.Body.String(), srv.otherAuditID) {
		t.Errorf("tenant search crossed into another tenant's audit event %q: %s", srv.otherAuditID, events.Body.String())
	}

	detail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/jobs/"+srv.otherJob.ID)
	if detail.Code != http.StatusNotFound {
		t.Errorf("cross-tenant detail status = %d, want %d; body = %s", detail.Code, http.StatusNotFound, detail.Body.String())
	}
	auditDetail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/audit_events/"+srv.otherAuditID)
	if auditDetail.Code != http.StatusNotFound {
		t.Errorf("cross-tenant audit detail status = %d, want %d; body = %s",
			auditDetail.Code, http.StatusNotFound, auditDetail.Body.String())
	}

	cancel := browserPost(t, srv.e, "/realms/default/api/admin/v1/jobs/"+srv.otherJob.ID+"/cancel")
	if cancel.Code != http.StatusNotFound {
		t.Errorf("cross-tenant cancel status = %d, want %d; body = %s", cancel.Code, http.StatusNotFound, cancel.Body.String())
	}
	got, err := srv.jobs.Get(t.Context(), srv.otherJob.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status == jobdomain.StatusCanceled {
		t.Fatalf("tenant route canceled another tenant's job %q", srv.otherJob.ID)
	}
}
