package server_http

// 主要ユースケース追跡: REQ-TENANCY-012。
//
// 製品の外部入口 (Register が組み立てるルート木、テナント解決ミドルウェア、
// 認証合成を含む) から、テナントのクォータ更新が Origin と CSRF トークンの検証を
// 通ることを確かめる。合成の側でクォータ経路だけが検査を外れていないかを見るため、
// ハンドラーを直接構築せずサーバー全体へ要求を投げる。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	quotaCsrfIssuer      = "http://idp.test"
	quotaCsrfTargetID    = "acme"
	quotaCsrfSeededUsers = 100
	quotaCsrfRequested   = 20000
)

// observedQuotaRepository は SetQuota の呼び出し回数を数える。403 を書いた後に保存へ
// 進む誤実装は応答からは見えないため、ポートの呼び出しそのものを観測する。
type observedQuotaRepository struct {
	*tenancymemory.QuotaRepository
	setQuotaCalls atomic.Int32
}

func (r *observedQuotaRepository) SetQuota(ctx context.Context, tenantID string, quota *tenancydomain.TenantQuota) error {
	r.setQuotaCalls.Add(1)
	return r.QuotaRepository.SetQuota(ctx, tenantID, quota)
}

type quotaCsrfServer struct {
	e      *echo.Echo
	quotas *observedQuotaRepository
}

func newQuotaCsrfServer(t *testing.T) *quotaCsrfServer {
	t.Helper()
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default",
			Status: tenancydomain.TenantStatusActive, CreatedAt: now,
		},
		{
			ID: quotaCsrfTargetID, Realm: quotaCsrfTargetID, DisplayName: "Acme",
			Status: tenancydomain.TenantStatusActive, CreatedAt: now,
		},
	} {
		if err := tenants.Save(t.Context(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	users := usermemory.NewUserRepository()
	users.Seed(controlPlaneTestUser("ops", tenancydomain.DefaultTenantID, "system_admin"))

	quotas := &observedQuotaRepository{QuotaRepository: tenancymemory.NewQuotaRepository()}
	seeded := quotaCsrfSeededUsers
	if err := quotas.QuotaRepository.SetQuota(t.Context(), quotaCsrfTargetID, &tenancydomain.TenantQuota{Users: &seeded}); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	Register(e, Deps{
		Issuer: quotaCsrfIssuer, Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, UserRepo: users, AuthnResolver: &fixedAuthnResolver{sub: "ops"},
		Tenancy: tenancy.Module{TenantRepo: tenants, QuotaRepo: quotas},
	})
	return &quotaCsrfServer{e: e, quotas: quotas}
}

func (s *quotaCsrfServer) updateQuota(mutate func(*http.Request)) *httptest.ResponseRecorder {
	body := strings.NewReader(`{"users":` + strconv.Itoa(quotaCsrfRequested) + `}`)
	req := httptest.NewRequest(
		http.MethodPut,
		"/realms/"+tenancydomain.DefaultRealm+"/api/admin/v1/tenants/"+quotaCsrfTargetID+"/quota",
		body,
	)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

// storedUsers は保存済みのユーザー上限を返す。上限が未設定なら -1 を返す。
func (s *quotaCsrfServer) storedUsers(t *testing.T) int {
	t.Helper()
	quota, err := s.quotas.GetQuota(t.Context(), quotaCsrfTargetID)
	if err != nil {
		t.Fatal(err)
	}
	if quota == nil || quota.Users == nil {
		return -1
	}
	return *quota.Users
}

func withSystemConsoleCSRF(token string) func(*http.Request) {
	return func(req *http.Request) {
		req.Header.Set("Origin", quotaCsrfIssuer)
		req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: token})
		req.Header.Set(support.CSRFHeader, token)
	}
}

// REQ-TENANCY-012: セッション Cookie だけでは、そのクォータ変更が管理 UI から出た
// ことを証明できない。CSRF トークンを伴わない要求は 403 で拒否され、保存ポートは
// 呼ばれず、保存済みのクォータも利用量も変わらない。
func TestUpdateTenantQuotaRejectsCookieSessionWithoutCSRF(t *testing.T) {
	srv := newQuotaCsrfServer(t)

	rec := srv.updateQuota(nil)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if calls := srv.quotas.setQuotaCalls.Load(); calls != 0 {
		t.Errorf("QuotaRepo.SetQuota calls = %d, want 0", calls)
	}
	if users := srv.storedUsers(t); users != quotaCsrfSeededUsers {
		t.Errorf("stored quota.users = %d, want %d (the refusal changed stored state)", users, quotaCsrfSeededUsers)
	}
	usage, err := srv.quotas.GetUsage(t.Context(), quotaCsrfTargetID)
	if err != nil {
		t.Fatal(err)
	}
	if usage.Users != 0 {
		t.Errorf("usage.users = %d, want 0 (the refusal changed recorded usage)", usage.Users)
	}
}

// REQ-TENANCY-012: 正しい Origin と double-submit された CSRF トークンを持つ
// システムコンソールのセッションは、これまでどおりクォータ上限を変更できる。
func TestUpdateTenantQuotaAcceptsSystemConsoleSessionWithCSRF(t *testing.T) {
	srv := newQuotaCsrfServer(t)

	rec := srv.updateQuota(withSystemConsoleCSRF("console-token"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if users := srv.storedUsers(t); users != quotaCsrfRequested {
		t.Errorf("stored quota.users = %d, want %d", users, quotaCsrfRequested)
	}
}

// Tenancy が登録する状態変更ルートの棚卸しを、散文ではなくテストとして固定する。
// どのルートも周囲資格情報の要求を副作用より前に拒否するので、リポジトリを一切
// 与えないサーバーでも 403 が返る。新しい状態変更ルートを検査なしで足すと、この
// 表に追加した時点で落ちる。
func TestTenancyStateChangingAdminRoutesVerifyBrowserRequests(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(t.Context(), &tenancydomain.Tenant{
		ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default",
		Status: tenancydomain.TenantStatusActive, CreatedAt: time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	users := usermemory.NewUserRepository()
	users.Seed(controlPlaneTestUser("ops", tenancydomain.DefaultTenantID, "system_admin"))

	e := echo.New()
	Register(e, Deps{
		Issuer: quotaCsrfIssuer, Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, UserRepo: users, AuthnResolver: &fixedAuthnResolver{sub: "ops"},
		Tenancy: tenancy.Module{TenantRepo: tenants},
	})

	const prefix = "/realms/" + tenancydomain.DefaultRealm
	for _, route := range []struct{ method, path string }{
		{http.MethodPatch, prefix + "/api/admin/v1/settings"},
		{http.MethodPut, prefix + "/api/admin/v1/tenant/user_attribute_schema"},
		{http.MethodPut, prefix + "/api/admin/v1/tenant/group_attribute_schema"},
		{http.MethodPut, prefix + "/api/admin/v1/tenant/branding"},
		{http.MethodPost, prefix + "/api/admin/v1/tenant/branding/assets/logo"},
		{http.MethodDelete, prefix + "/api/admin/v1/tenant/branding/assets/logo"},
		{http.MethodPut, prefix + "/api/admin/v1/tenant/notification_templates/password_reset/ja"},
		{http.MethodDelete, prefix + "/api/admin/v1/tenant/notification_templates/password_reset/ja"},
		{http.MethodPost, prefix + "/api/admin/v1/tenant/notification_templates/password_reset/ja/preview"},
		{http.MethodPost, prefix + "/api/admin/v1/tenant/notification_templates/password_reset/ja/test"},
		{http.MethodPost, prefix + "/api/admin/v1/tenants"},
		{http.MethodPatch, prefix + "/api/admin/v1/tenants/acme"},
		{http.MethodPut, prefix + "/api/admin/v1/tenants/acme/endpoint_style"},
		{http.MethodPost, prefix + "/api/admin/v1/tenants/acme/disable"},
		{http.MethodPost, prefix + "/api/admin/v1/tenants/acme/enable"},
		{http.MethodPut, prefix + "/api/admin/v1/tenants/acme/quota"},
	} {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, strings.NewReader("{}"))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
		})
	}
}
