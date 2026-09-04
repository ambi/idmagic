package handlers_http_test

// 主要ユースケース追跡: REQ-TENANCY-012。
//
// 制御面のテナント CRUD アダプタそのものの境界で、クォータ更新が
// Origin と CSRF トークンの検証を副作用より前に通すことを確かめる。
// サーバー合成やテナント解決ミドルウェアは挟まず、
// RegisterControlPlaneRoutes が組み立てる経路だけを対象にする。

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/handlers_http"

	"github.com/labstack/echo/v5"
)

const (
	quotaTestIssuer = "http://idp.test"
	quotaTestTenant = "acme"
	// quotaTestSeededUsers は要求より前に保存されているユーザー上限。拒否された要求が
	// これを書き換えていないことを、応答コードとは別に確かめるための基準値である。
	quotaTestSeededUsers = 100
	quotaTestRequested   = 20000
)

// observedQuotaRepository は SetQuota の呼び出し回数を数える。403 を書いた後に保存へ
// 進む誤実装は応答からは見えないため、ポートの呼び出しそのものを観測する。
type observedQuotaRepository struct {
	*memory.QuotaRepository
	setQuotaCalls atomic.Int32
}

func (r *observedQuotaRepository) SetQuota(ctx context.Context, tenantID string, quota *domain.TenantQuota) error {
	r.setQuotaCalls.Add(1)
	return r.QuotaRepository.SetQuota(ctx, tenantID, quota)
}

// staticIntrospector は Bearer / DPoP の資格情報を周囲資格情報ではない経路として
// 通すための最小の内省結果を返す。DPoP の送信者束縛は付けないので、この経路は
// Authorization ヘッダーの扱いだけを検証する。
type staticIntrospector struct {
	sub   string
	scope string
}

func (i staticIntrospector) IntrospectAccessToken(context.Context, string) (*oauthports.IntrospectionResult, error) {
	return &oauthports.IntrospectionResult{
		Active: true, Sub: i.sub, Scope: i.scope, Iat: time.Now().UTC().Unix(),
	}, nil
}

type quotaTestServer struct {
	e      *echo.Echo
	quotas *observedQuotaRepository
}

func newQuotaControlPlaneServer(t *testing.T) *quotaTestServer {
	t.Helper()
	now := time.Now().UTC()

	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range []*domain.Tenant{
		{
			ID: domain.DefaultTenantID, Realm: domain.DefaultRealm, DisplayName: "Default",
			Status: domain.TenantStatusActive, CreatedAt: now,
		},
		{
			ID: quotaTestTenant, Realm: quotaTestTenant, DisplayName: "Acme",
			Status: domain.TenantStatusActive, CreatedAt: now,
		},
	} {
		if err := tenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}

	userRepo := usermemory.NewUserRepository()
	actor := &userdomain.User{
		ID: "ops", PreferredUsername: "ops", TenantID: domain.DefaultTenantID,
		Roles: []string{"system_admin"}, CreatedAt: now, UpdatedAt: now,
	}
	userRepo.Seed(actor)

	quotas := &observedQuotaRepository{QuotaRepository: memory.NewQuotaRepository()}
	seeded := quotaTestSeededUsers
	if err := quotas.QuotaRepository.SetQuota(context.Background(), quotaTestTenant, &domain.TenantQuota{Users: &seeded}); err != nil {
		t.Fatal(err)
	}

	e := echo.New()
	tenancyhttp.RegisterControlPlaneRoutes(e.Group(""), tenancyhttp.Deps{
		Deps: support.Deps{
			Issuer: quotaTestIssuer, Contract: spec.CurrentRuntimeContract(), TenantRepo: tenantRepo,
		},
		Authenticator: &support.Authenticator{
			UserRepo: userRepo,
			AuthnResolver: &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
				UserID: actor.ID, AuthTime: now.Unix(), AMR: []string{"pwd"},
			}},
			TokenIntrospector: staticIntrospector{sub: actor.ID, scope: "idmagic.admin"},
		},
		TenantRepo: tenantRepo,
		UserRepo:   userRepo,
		QuotaRepo:  quotas,
	})
	return &quotaTestServer{e: e, quotas: quotas}
}

func (s *quotaTestServer) updateQuota(mutate func(*http.Request)) *httptest.ResponseRecorder {
	return s.putQuotaBody(`{"users":`+strconv.Itoa(quotaTestRequested)+`}`, mutate)
}

func (s *quotaTestServer) putQuotaBody(body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/api/admin/v1/tenants/"+quotaTestTenant+"/quota", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

// storedUsers は保存済みのユーザー上限を返す。上限が未設定なら -1 を返す。
func (s *quotaTestServer) storedUsers(t *testing.T) int {
	t.Helper()
	quota, err := s.quotas.GetQuota(t.Context(), quotaTestTenant)
	if err != nil {
		t.Fatal(err)
	}
	if quota == nil || quota.Users == nil {
		return -1
	}
	return *quota.Users
}

func withBrowserCredentials(origin, cookie, header string) func(*http.Request) {
	return func(req *http.Request) {
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: cookie})
		}
		if header != "" {
			req.Header.Set(support.CSRFHeader, header)
		}
	}
}

// REQ-TENANCY-012: クォータ上限の変更は制御面の状態変更であり、Cookie セッションから
// 呼ぶ場合は Origin と CSRF トークンの一致を要求する。拒否は応答コードだけでなく、
// 保存ポートが呼ばれないことと保存済みの値が変わらないことまでを含む。
func TestUpdateTenantQuotaRefusesCookieSessionWithoutCSRF(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request func(*http.Request)
	}{
		{name: "no origin and no token", request: nil},
		{name: "mismatched origin", request: withBrowserCredentials("http://evil.test", "token", "token")},
		{name: "missing csrf cookie", request: withBrowserCredentials(quotaTestIssuer, "", "token")},
		{name: "missing csrf header", request: withBrowserCredentials(quotaTestIssuer, "token", "")},
		{name: "cookie and header disagree", request: withBrowserCredentials(quotaTestIssuer, "token-a", "token-b")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newQuotaControlPlaneServer(t)

			rec := srv.updateQuota(tc.request)

			if rec.Code != http.StatusForbidden {
				t.Errorf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
			}
			if calls := srv.quotas.setQuotaCalls.Load(); calls != 0 {
				t.Errorf("QuotaRepo.SetQuota calls = %d, want 0", calls)
			}
			if users := srv.storedUsers(t); users != quotaTestSeededUsers {
				t.Errorf("stored quota.users = %d, want %d (the refusal changed stored state)", users, quotaTestSeededUsers)
			}
		})
	}
}

// REQ-TENANCY-012: 拒否検査は要求本文のデコードより前に立つ。壊れた本文を送っても
// 400 ではなく CSRF の 403 が返ることで、周囲資格情報を持たない呼び出し元が本文の
// 妥当性を観測できないことを固定する。検査を本文デコードの後ろへ移す実装は、
// 副作用を防いだままこの区別だけを失うため、状態の観測だけでは検出できない。
func TestUpdateTenantQuotaRefusesBeforeDecodingTheBody(t *testing.T) {
	srv := newQuotaControlPlaneServer(t)

	rec := srv.putQuotaBody(`{"users":`, nil)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid_origin") {
		t.Errorf("body = %s, want the origin refusal rather than a body-validation refusal", rec.Body.String())
	}
}

// REQ-TENANCY-012: 正しい Origin と double-submit された CSRF トークンを伴う
// システムコンソールの要求は、これまでどおりクォータ上限を変更できる。
func TestUpdateTenantQuotaAcceptsMatchingOriginAndCSRFToken(t *testing.T) {
	srv := newQuotaControlPlaneServer(t)

	rec := srv.updateQuota(withBrowserCredentials(quotaTestIssuer, "token-match", "token-match"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if users := srv.storedUsers(t); users != quotaTestRequested {
		t.Errorf("stored quota.users = %d, want %d", users, quotaTestRequested)
	}
}

// REQ-TENANCY-012: Authorization ヘッダーで渡す Bearer と DPoP の資格情報は
// 周囲資格情報ではないため、Cookie の CSRF 検査の対象外である。自動化から
// クォータを更新する経路をこの変更で塞がないことを固定する。
func TestUpdateTenantQuotaAcceptsNonAmbientCredentials(t *testing.T) {
	for _, scheme := range []string{"Bearer", "DPoP"} {
		t.Run(scheme, func(t *testing.T) {
			srv := newQuotaControlPlaneServer(t)

			rec := srv.updateQuota(func(req *http.Request) {
				req.Header.Set("Authorization", scheme+" access-token")
			})

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
			}
			if users := srv.storedUsers(t); users != quotaTestRequested {
				t.Errorf("stored quota.users = %d, want %d", users, quotaTestRequested)
			}
		})
	}
}
