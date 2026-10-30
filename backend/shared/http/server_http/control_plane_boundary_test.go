package server_http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/audit"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/datakeys"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_cleartext"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/signingkeys"

	"github.com/ambi/idmagic/backend/tenancy"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

type observedTenantRepository struct {
	tenantports.TenantRepository
	findAllCalls atomic.Int32
}

func (r *observedTenantRepository) FindAll(ctx context.Context) ([]*tenancydomain.Tenant, error) {
	r.findAllCalls.Add(1)
	return r.TenantRepository.FindAll(ctx)
}

type controlPlaneBoundaryServer struct {
	e              *echo.Echo
	tenants        *observedTenantRepository
	jobs           jobports.JobRepository
	otherJob       *jobdomain.Job
	dataKeys       datakeys.Module
	events         *[]spec.DomainEvent
	otherAuditID   string
	controlAuditID string
}

func newControlPlaneBoundaryServer(t *testing.T, actor *userdomain.User) *controlPlaneBoundaryServer {
	t.Helper()
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	tenantStore := tenancy.NewMemoryModule().TenantRepo
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: "acme", Realm: "acme", DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
	} {
		if err := tenantStore.Save(t.Context(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	tenants := &observedTenantRepository{TenantRepository: tenantStore}

	users := idmanagement.NewMemoryModule().UserRepo
	_ = users.Save(context.Background(), actor)
	resolver := &fixedAuthnResolver{sub: actor.ID}

	keyStore, err := signingkeys.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	masterKey, err := envelope_cleartext.NewCleartextMasterKeyProvider()
	if err != nil {
		t.Fatal(err)
	}
	crypto := envelope_crypto.NewTinkEnvelopeCrypto(masterKey)
	// DEK は 2 テナントに置き、acme だけ回転させる。健全性一覧がテナントごとの版を
	// 取り違えずに返すかを、版の違いで判定できるようにするためである。
	dataKeyModule := datakeys.NewMemoryModule(crypto)
	for _, tenantID := range []string{tenancydomain.DefaultTenantID, "acme"} {
		if _, err := dataKeyModule.BootstrapTenantDataKey(t.Context(), tenantID, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := dataKeyModule.RotateTenantDataKey(t.Context(), "acme", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	dataKeyModule.Migrators.Register("fixture", dataKeyMigrator{})

	jobStore := jobs.NewJobRepository()
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
	auditStore := audit.NewAuditEventStore(0)
	for _, rec := range []*auditports.AuditEventRecord{
		{ID: "audit-acme", TenantID: "acme", Type: "UserCreated", OccurredAt: now},
		{ID: "audit-control", TenantID: tenancydomain.DefaultTenantID, Type: "UserCreated", OccurredAt: now.Add(time.Minute)},
	} {
		if err := auditStore.Append(t.Context(), rec); err != nil {
			t.Fatal(err)
		}
	}

	events := &[]spec.DomainEvent{}
	emit := func(event spec.DomainEvent) { *events = append(*events, event) }
	e := echo.New()
	Register(e, Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, UserRepo: users, AuthnResolver: resolver, Emit: emit,
		SigningKeys: signingkeys.Module{KeyStore: keyStore},
		DataKeys:    dataKeyModule,
		Jobs:        jobs.Module{Repo: jobStore, Enqueuer: jobs.NewEnqueuer(jobStore, tenancy.NewMemoryModule().QuotaRepo, emit)},
		Audit:       audit.Module{AuditEventRepo: auditStore},
	})
	return &controlPlaneBoundaryServer{
		e: e, tenants: tenants, jobs: jobStore, otherJob: otherJob, dataKeys: dataKeyModule, events: events,
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
	if !strings.Contains(rec.Body.String(), "access_denied") {
		t.Errorf("refusal body is not access_denied: %s", rec.Body.String())
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

//spec:covers REQ-SIGNINGKEYS-009: 制御面テナント外の system_admin は、署名鍵ヘルスも横断収集も観測できない。
func TestControlPlaneSigningKeyHealthRejectsSystemAdminOutsideControlPlaneTenant(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin"))
	rec := getControlPlaneBoundary(srv.e, "/realms/acme/api/admin/v1/keys/health")
	assertHealthRefusal(t, rec, srv.tenants.findAllCalls.Load())
}

//spec:covers REQ-DATAKEYS-006, EX-DATAKEYS-006-03: 制御面テナント外の system_admin は 403 access_denied で拒否され、本文に他テナントの識別子も DEK の状態も無く、横断収集の FindAll も呼ばれない。
func TestControlPlaneDataKeyHealthRejectsSystemAdminOutsideControlPlaneTenant(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin"))
	rec := getControlPlaneBoundary(srv.e, "/realms/acme/api/admin/v1/data-keys/health")
	assertHealthRefusal(t, rec, srv.tenants.findAllCalls.Load())
}

//spec:covers EX-DATAKEYS-006-02: 制御面テナントに所属していても system_admin を持たない主体は 403 access_denied で拒否され、本文に DEK の状態が無く、横断収集の FindAll も呼ばれない。
func TestControlPlaneDataKeyHealthRejectsControlPlaneMemberWithoutSystemAdmin(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-admin", tenancydomain.DefaultTenantID, "admin"))
	rec := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/data-keys/health")
	assertHealthRefusal(t, rec, srv.tenants.findAllCalls.Load())
}

//spec:covers EX-DATAKEYS-006-01: 制御面テナントの system_admin には、DEK を持つ各テナントの active_version、status、プロバイダーへの到達性が返り、wrapped_dek と master_key_id は応答のどこにも現れない。
func TestControlPlaneDataKeyHealthListsEveryTenantWithoutKeyMaterial(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))
	rec := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/data-keys/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Tenants []map[string]any `json:"tenants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}

	wantActiveVersion := map[string]float64{tenancydomain.DefaultTenantID: 1, "acme": 2}
	disclosed := map[string]bool{"tenant_id": true, "active_version": true, "status": true, "provider": true, "provider_reachable": true, "rotated_at": true}
	seen := map[string]bool{}
	for _, entry := range body.Tenants {
		tenantID, _ := entry["tenant_id"].(string)
		seen[tenantID] = true
		if entry["active_version"] != wantActiveVersion[tenantID] || entry["status"] != "active" || entry["provider_reachable"] != true {
			t.Errorf("tenant %q health = %v, want active_version %v, status active, provider reachable", tenantID, entry, wantActiveVersion[tenantID])
		}
		for field := range entry {
			if !disclosed[field] {
				t.Errorf("tenant %q health discloses %q: %v", tenantID, field, entry)
			}
		}
	}
	for tenantID := range wantActiveVersion {
		if !seen[tenantID] {
			t.Errorf("health omits tenant %q: %s", tenantID, rec.Body.String())
		}
	}

	// 項目名に出なくても、鍵素材の値が他の項目へ紛れ込めば漏洩である。保存値そのものを本文から探す。
	for tenantID := range wantActiveVersion {
		keys, err := srv.dataKeys.Repository.ListAll(t.Context(), tenantID)
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range keys {
			for _, material := range []string{base64.StdEncoding.EncodeToString(key.WrappedDEK), key.MasterKeyID} {
				if strings.Contains(rec.Body.String(), material) {
					t.Errorf("health body carries key material of tenant %q version %d", tenantID, key.Version)
				}
			}
		}
	}
}

// browserPost はブラウザー経路の変更操作を、CSRF トークンと Origin を揃えて送る。
func browserPost(t *testing.T, e *echo.Echo, path string) *httptest.ResponseRecorder {
	t.Helper()
	realmPrefix, _, _ := strings.Cut(path, "/api/")
	account := getControlPlaneBoundary(e, realmPrefix+"/api/auth/account")
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

//spec:covers REQ-JOBS-015 / EX-JOBS-015-01: 制御面主体はシステム経路で admin ロールなしに一覧、詳細、取消しを横断できる。
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

//spec:covers EX-JOBS-015-03: ブラウザーからの要求であることを証明できない取消しは、Job を変えずに拒否される。
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

//spec:covers REQ-AUDIT-007 / EX-AUDIT-007-01: 制御面主体はシステム経路で全テナントの監査イベントを検索、参照、エクスポートできる。
func TestSystemAuditEventsSpanEveryTenantForControlPlaneActor(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	list := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/audit-events")
	if list.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", list.Code, list.Body.String())
	}
	for _, want := range []string{srv.otherAuditID, srv.controlAuditID} {
		if !strings.Contains(list.Body.String(), want) {
			t.Fatalf("cross-tenant search omitted audit event %q: %s", want, list.Body.String())
		}
	}

	export := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/audit-events/export")
	if export.Code != http.StatusOK {
		t.Fatalf("export status = %d, body = %s", export.Code, export.Body.String())
	}
	if !strings.Contains(export.Body.String(), srv.otherAuditID) {
		t.Fatalf("cross-tenant export omitted audit event %q: %s", srv.otherAuditID, export.Body.String())
	}

	detail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/system/audit-events/"+srv.otherAuditID)
	if detail.Code != http.StatusOK {
		t.Fatalf("detail status = %d, body = %s", detail.Code, detail.Body.String())
	}
}

// どのテナントの記録も観測できない。
//
//spec:covers EX-AUDIT-007-02, EX-JOBS-015-02, EX-SYSTEM-020-03: 制御面主体でない実行者はシステム経路から
func TestSystemRoutesRefuseNonControlPlaneActor(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin", "admin"))

	for _, path := range []string{
		"/realms/acme/api/admin/v1/system/audit-events",
		"/realms/acme/api/admin/v1/system/audit-events/export",
		"/realms/acme/api/admin/v1/system/audit-events/" + srv.otherAuditID,
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
// 要求先テナントへ閉じる。横断を求める入力を添えても範囲は変わらない。
//
//spec:covers EX-JOBS-012-04, EX-JOBS-013-05, EX-SYSTEM-020-01: テナント管理経路は制御面主体に対しても
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

	events := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/audit-events?all_tenants=true")
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
	auditDetail := getControlPlaneBoundary(srv.e, "/realms/default/api/admin/v1/audit-events/"+srv.otherAuditID)
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

// dataKeyMigrator は再暗号化の投入先を観測するための登録である。投入そのものは本番の Enqueuer を通す。
type dataKeyMigrator struct{}

func (dataKeyMigrator) ReencryptBatch(context.Context, string, int, int) (int, error) { return 0, nil }

func (dataKeyMigrator) PendingCount(context.Context, string, int) (int, error) { return 0, nil }

func (s *controlPlaneBoundaryServer) activeDataKeyVersion(t *testing.T, tenantID string) int {
	t.Helper()
	key, err := s.dataKeys.Repository.FindActive(t.Context(), tenantID)
	if err != nil {
		t.Fatal(err)
	}
	return key.Version
}

func (s *controlPlaneBoundaryServer) reencryptionJobs(t *testing.T, tenantID string) []*jobdomain.Job {
	t.Helper()
	found, err := s.jobs.ListByTenantAndKinds(t.Context(), tenantID, []jobdomain.JobKind{jobdomain.KindDataKeyReencryption}, 10)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

//spec:covers REQ-DATAKEYS-008, REQ-DATAKEYS-002: 制御面主体の要求で、パスが指すテナントだけがローテーションされ、応答に鍵素材がなく、操作者付きのイベントと再暗号化のジョブが残る。
//verify:operation RotateTenantDataKey: 200 の本文の tenant_id、active_version、previous_version と、保存後の active のバージョンを照合する。
func TestControlPlaneRotatesTenantDataKeyWithAuditAndReencryption(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	rec := browserPost(t, srv.e, "/realms/default/api/admin/v1/tenants/acme/data-key/rotate")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"tenant_id": "acme", "active_version": float64(3), "previous_version": float64(2)}
	if !maps.Equal(body, want) {
		t.Fatalf("body = %v, want %v", body, want)
	}
	if got := srv.activeDataKeyVersion(t, "acme"); got != 3 {
		t.Fatalf("acme active version = %d, want 3", got)
	}
	if got := srv.activeDataKeyVersion(t, tenancydomain.DefaultTenantID); got != 1 {
		t.Fatalf("requesting tenant was rotated: active version = %d, want 1", got)
	}
	if got := srv.reencryptionJobs(t, "acme"); len(got) != 1 {
		t.Fatalf("acme reencryption jobs = %d, want 1", len(got))
	}

	var rotated map[string]any
	for _, event := range *srv.events {
		if event.EventType() == "DataEncryptionKeyRotated" {
			payload, err := json.Marshal(event)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(payload, &rotated); err != nil {
				t.Fatal(err)
			}
		}
	}
	if rotated["actorUserId"] != "control-operator" || rotated["tenantId"] != "acme" || rotated["newVersion"] != float64(3) {
		t.Fatalf("DataEncryptionKeyRotated payload = %v", rotated)
	}
}

//spec:covers REQ-DATAKEYS-008, REQ-DATAKEYS-009: テナントの system_admin は、自テナントを指しても 403 access_denied で拒否され、バージョンは変わらない。
func TestControlPlaneDataKeyActionsRefuseActorOutsideControlPlane(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("acme-operator", "acme", "system_admin", "admin"))

	for _, path := range []string{
		"/realms/acme/api/admin/v1/tenants/acme/data-key/rotate",
		"/realms/acme/api/admin/v1/tenants/acme/data-key/versions/1/disable",
	} {
		rec := browserPost(t, srv.e, path)
		if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "access_denied") {
			t.Errorf("%s: status = %d, body = %s", path, rec.Code, rec.Body.String())
		}
	}
	if got := srv.activeDataKeyVersion(t, "acme"); got != 2 {
		t.Fatalf("active version = %d, want 2", got)
	}
	retired, err := srv.dataKeys.Repository.FindByVersion(t.Context(), "acme", 1)
	if err != nil {
		t.Fatal(err)
	}
	if retired.Status != "retiring" {
		t.Fatalf("version 1 status = %q, want retiring", retired.Status)
	}
}

//spec:covers REQ-DATAKEYS-008, REQ-DATAKEYS-009: ブラウザーからの要求であることを証明できない操作は、DEK を変えずに拒否される。
func TestControlPlaneDataKeyActionsRefuseWithoutBrowserProof(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))

	for _, path := range []string{
		"/realms/default/api/admin/v1/tenants/acme/data-key/rotate",
		"/realms/default/api/admin/v1/tenants/acme/data-key/versions/1/disable",
	} {
		rec := httptest.NewRecorder()
		srv.e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, http.NoBody))
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403; body = %s", path, rec.Code, rec.Body.String())
		}
	}
	if got := srv.activeDataKeyVersion(t, "acme"); got != 2 {
		t.Fatalf("active version = %d, want 2", got)
	}
}

//spec:covers REQ-DATAKEYS-009, REQ-DATAKEYS-003, REQ-DATAKEYS-004: retiring のバージョンは 204 で disabled になり、保持した DEK も捨てられて復号が拒否される。active のバージョンの無効化は 400 で状態を変えない。
//verify:operation DisableTenantDataKey: 204 と保存後の復号の拒否、active の指定への 400 invalid_request を照合する。
func TestControlPlaneDisablesRetiringDataKeyAndRefusesActive(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))
	cipher := srv.dataKeys.FieldCipher()
	ctx := t.Context()
	version, ciphertext, err := cipher.Encrypt(ctx, "acme", "ctx", "table", "record", "field", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if rec := browserPost(t, srv.e, "/realms/default/api/admin/v1/tenants/acme/data-key/rotate"); rec.Code != http.StatusOK {
		t.Fatalf("rotate status = %d, body = %s", rec.Code, rec.Body.String())
	}
	// 復号して DEK を保持させる。保持を捨てなければ、無効化の後も復号できてしまう。
	if _, err := cipher.Decrypt(ctx, "acme", "ctx", "table", "record", "field", version, ciphertext); err != nil {
		t.Fatalf("decrypt under retiring version: %v", err)
	}

	active := srv.activeDataKeyVersion(t, "acme")
	refused := browserPost(t, srv.e, fmt.Sprintf("/realms/default/api/admin/v1/tenants/acme/data-key/versions/%d/disable", active))
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "invalid_request") {
		t.Fatalf("disable active: status = %d, body = %s", refused.Code, refused.Body.String())
	}
	if got := srv.activeDataKeyVersion(t, "acme"); got != active {
		t.Fatalf("refused disable changed active version to %d", got)
	}

	rec := browserPost(t, srv.e, fmt.Sprintf("/realms/default/api/admin/v1/tenants/acme/data-key/versions/%d/disable", version))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("disable status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if _, err := cipher.Decrypt(ctx, "acme", "ctx", "table", "record", "field", version, ciphertext); !errors.Is(err, envelope_crypto.ErrDataKeyUnavailable) {
		t.Fatalf("decrypt after disable error = %v, want ErrDataKeyUnavailable", err)
	}
	var actor string
	for _, event := range *srv.events {
		if disabled, err := json.Marshal(event); err == nil && event.EventType() == "DataEncryptionKeyDisabled" {
			var payload map[string]any
			if err := json.Unmarshal(disabled, &payload); err != nil {
				t.Fatal(err)
			}
			actor, _ = payload["actorUserId"].(string)
		}
	}
	if actor != "control-operator" {
		t.Fatalf("DataEncryptionKeyDisabled actor = %q", actor)
	}
}

//spec:covers REQ-DATAKEYS-008, REQ-DATAKEYS-009: 存在しないテナントとバージョンは 404、DEK を持たないテナントは 400 で、DEK を作らない。
//verify:operation RotateTenantDataKey: 存在しないテナントへの 404 tenant_not_found と、DEK のないテナントへの 400 invalid_request の本文を照合する。
//verify:operation DisableTenantDataKey: 存在しないテナントとバージョンへの 404 と、不正なバージョンへの 400 の本文を照合する。
func TestControlPlaneDataKeyActionsMapMissingTargets(t *testing.T) {
	srv := newControlPlaneBoundaryServer(t, controlPlaneTestUser("control-operator", tenancydomain.DefaultTenantID, "system_admin"))
	now := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	if err := srv.tenants.Save(t.Context(), &tenancydomain.Tenant{ID: "empty", Realm: "empty", DisplayName: "Empty", Status: tenancydomain.TenantStatusActive, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		path   string
		status int
		code   string
	}{
		{"/realms/default/api/admin/v1/tenants/missing/data-key/rotate", http.StatusNotFound, "tenant_not_found"},
		{"/realms/default/api/admin/v1/tenants/missing/data-key/versions/1/disable", http.StatusNotFound, "tenant_not_found"},
		{"/realms/default/api/admin/v1/tenants/acme/data-key/versions/99/disable", http.StatusNotFound, "data_key_version_not_found"},
		{"/realms/default/api/admin/v1/tenants/acme/data-key/versions/nope/disable", http.StatusBadRequest, "invalid_request"},
		{"/realms/default/api/admin/v1/tenants/empty/data-key/rotate", http.StatusBadRequest, "invalid_request"},
	} {
		rec := browserPost(t, srv.e, tc.path)
		if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.code) {
			t.Errorf("%s: status = %d, body = %s; want %d %s", tc.path, rec.Code, rec.Body.String(), tc.status, tc.code)
		}
	}
	if _, err := srv.dataKeys.Repository.FindActive(t.Context(), "empty"); err == nil {
		t.Fatal("rotate created a DEK for a tenant without one")
	}
}
