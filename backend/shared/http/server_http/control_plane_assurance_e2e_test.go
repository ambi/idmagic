package server_http_test

// 制御面の操作は MFA で成立した認証を求め、変更は直近 5 分の MFA の認証を求める
// (REQ-SYSTEM-033)。長時間残ったトークンや、パスワードだけ、記憶済みの端末だけの
// 認証では、全テナントの読み出しもテナントの停止もできないことを HTTP 境界で確かめる。

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

const controlPlaneTargetTenant = "acme-control"

// controlPlaneToken は、system_admin の alice が amr と auth_time の認証で得たアクセストークンを返す。
func controlPlaneToken(t *testing.T, f *tiFixture, amr []string, authenticatedAt time.Time) string {
	t.Helper()
	ctx := testing_tenant.Default(t.Context())
	user, err := f.users.FindBySub(ctx, tiUserID)
	if err != nil || user == nil {
		t.Fatalf("alice が見つからない: %v", err)
	}
	user.Roles = []string{"system_admin"}
	if err := f.users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	code := f.authorizationCode(t, url.Values{"scope": {"openid profile offline_access idmagic.admin"}})
	record, err := f.codes.Find(ctx, code)
	if err != nil || record == nil {
		t.Fatalf("認可コードの記録が見つからない: %v", err)
	}
	record.AuthTime, record.AMR = authenticatedAt.Unix(), amr
	if err := f.codes.Save(ctx, record); err != nil {
		t.Fatal(err)
	}
	return f.mustToken(t, f.codeExchangeForm(code, nil)).AccessToken
}

func seedControlPlaneTarget(t *testing.T, f *tiFixture) {
	t.Helper()
	if err := f.tenants.Save(t.Context(), &tenancydomain.Tenant{
		ID: controlPlaneTargetTenant, Realm: controlPlaneTargetTenant, Status: tenancydomain.TenantStatusActive,
	}); err != nil {
		t.Fatal(err)
	}
}

func callSystemAPI(t *testing.T, f *tiFixture, method, path, token string) (int, http.Header, string) {
	t.Helper()
	request, err := http.NewRequest(method, f.base+path, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)
	return response.StatusCode, response.Header, string(body)
}

func assertInsufficientUserAuthentication(t *testing.T, status int, header http.Header, body string, wantMaxAge bool) {
	t.Helper()
	if status != http.StatusUnauthorized || !strings.Contains(body, "urn:idmagic:error:insufficient_user_authentication") {
		t.Fatalf("status=%d body=%s, want 401 insufficient_user_authentication", status, body)
	}
	challenge := header.Get("WWW-Authenticate")
	if !strings.Contains(challenge, `error="insufficient_user_authentication"`) || !strings.Contains(challenge, `acr_values="urn:idmagic:acr:mfa"`) {
		t.Fatalf("WWW-Authenticate=%q, want insufficient_user_authentication と MFA の acr_values", challenge)
	}
	if got := strings.Contains(challenge, `max_age="300"`); got != wantMaxAge {
		t.Fatalf("WWW-Authenticate=%q, max_age の有無=%v, want %v", challenge, got, wantMaxAge)
	}
}

//spec:covers REQ-SYSTEM-033: パスワードだけ、記憶済みの端末だけの認証のトークンではシステムの経路の読み出しを MFA の acr_values で拒否し、時間が経っていても MFA で成立した認証のトークンでは読めることを固定する。
//verify:operation ListTenants: MFA のない認証には 401 insufficient_user_authentication の本文と MFA の acr_values を返し、MFA の認証には 200 を返す。
//verify:operation ListTenantKeyHealth: MFA のない認証には 401 insufficient_user_authentication の本文と MFA の acr_values を返し、MFA の認証には 200 を返す。
//verify:operation ListTenantDataKeyHealth: MFA のない認証には 401 insufficient_user_authentication の本文と MFA の acr_values を返し、MFA の認証には 200 を返す。
func TestControlPlaneReadRequiresMfa(t *testing.T) {
	f := newTokenIssuanceFixture(t)
	anHourAgo := time.Now().Add(-time.Hour)
	for _, path := range []string{"/api/admin/v1/tenants", "/api/admin/v1/keys/health", "/api/admin/v1/data-keys/health"} {
		t.Run(path, func(t *testing.T) {
			for _, tc := range []struct {
				name string
				amr  []string
			}{
				{"パスワードだけ", []string{"pwd"}},
				{"記憶済みの端末だけ", []string{"pwd", "tdev"}},
			} {
				status, header, body := callSystemAPI(t, f, http.MethodGet, path, controlPlaneToken(t, f, tc.amr, time.Now()))
				assertInsufficientUserAuthentication(t, status, header, body, false)
			}
			if status, _, body := callSystemAPI(t, f, http.MethodGet, path, controlPlaneToken(t, f, []string{"pwd", "otp"}, anHourAgo)); status != http.StatusOK {
				t.Fatalf("MFA の認証の読み出し status=%d body=%s, want 200", status, body)
			}
		})
	}
}

//spec:covers REQ-SYSTEM-033: 5 分より前の MFA やパスワードだけの認証のトークンではテナントの無効化を max_age 付きで拒否してテナントを Active に残し、直近の MFA の認証のトークンでは無効化できることを固定する。
func TestControlPlaneChangeRequiresRecentMfa(t *testing.T) {
	f := newTokenIssuanceFixture(t)
	seedControlPlaneTarget(t, f)
	disable := "/api/admin/v1/tenants/" + controlPlaneTargetTenant + "/disable"

	status, header, body := callSystemAPI(t, f, http.MethodPost, disable, controlPlaneToken(t, f, []string{"pwd", "otp"}, time.Now().Add(-time.Hour)))
	assertInsufficientUserAuthentication(t, status, header, body, true)
	status, header, body = callSystemAPI(t, f, http.MethodPost, disable, controlPlaneToken(t, f, []string{"pwd"}, time.Now()))
	assertInsufficientUserAuthentication(t, status, header, body, true)
	tenant, err := f.tenants.FindByID(t.Context(), controlPlaneTargetTenant)
	if err != nil || tenant == nil || tenant.Status != tenancydomain.TenantStatusActive {
		t.Fatalf("拒否の後のテナント=%+v err=%v, want Active のまま", tenant, err)
	}

	if status, _, body := callSystemAPI(t, f, http.MethodPost, disable, controlPlaneToken(t, f, []string{"pwd", "otp"}, time.Now())); status != http.StatusNoContent {
		t.Fatalf("直近の MFA の認証の無効化 status=%d body=%s, want 204", status, body)
	}
	tenant, err = f.tenants.FindByID(t.Context(), controlPlaneTargetTenant)
	if err != nil || tenant == nil || tenant.Status != tenancydomain.TenantStatusDisabled {
		t.Fatalf("無効化の後のテナント=%+v err=%v, want Disabled", tenant, err)
	}
}
