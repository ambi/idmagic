package support_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// authTestRealmAPI は、テストのリクエスト先レルムの IdMagic API (レルムの発行者識別子) である。
// account スコープのトークンはこの値を audience に持たなければ受けられない。
const authTestRealmAPI = "https://idp.test/realms/acme"

// withAuthTestRealm は、テナント middleware が組み立てるのと同じレルムの文脈を req に載せる。
func withAuthTestRealm(req *http.Request) *http.Request {
	return req.WithContext(tenancy.WithTenant(req.Context(), &tenancydomain.Tenant{ID: "acme"}, authTestRealmAPI, "/realms/acme"))
}

// authTestAccessTokens は、イントロスペクションの結果を固定したアクセストークンの検証器である。
type authTestAccessTokens struct {
	token *AccessToken
}

func (f authTestAccessTokens) IntrospectAccessToken(context.Context, string) (*AccessToken, error) {
	return f.token, nil
}

func (authTestAccessTokens) VerifyDPoPProof(context.Context, DPoPProof) (string, error) {
	return "", errors.New("authTestAccessTokens does not verify DPoP proofs")
}

type authTestApiTokens struct {
	principal ApiTokenPrincipal
	err       error
}

func (f authTestApiTokens) AuthenticateApiToken(context.Context, string) (ApiTokenPrincipal, error) {
	return f.principal, f.err
}

// wi-275: account resource server の route と最小 scope の正準対応。
func TestRequiredAccountScope(t *testing.T) {
	for _, tc := range []struct {
		method, path, scope string
		allowed             bool
	}{
		{http.MethodGet, "/realms/acme/api/account/v1/profile", "account:read", true},
		{http.MethodPatch, "/realms/acme/api/account/v1/profile", "account:write", true},
		{http.MethodPost, "/realms/acme/api/account/v1/mfa/totp/remove", "account:mfa:write", true},
		{http.MethodPost, "/realms/acme/api/account/v1/sessions/s1/revoke", "account:sessions:write", true},
		{http.MethodPost, "/realms/acme/api/account/v1/consents/c1/revoke", "account:consents:write", true},
		{http.MethodPost, "/realms/acme/api/auth/change-password", "account:password:write", true},
		{http.MethodPost, "/realms/acme/api/account/v1/step-up/start", "", false},
		{http.MethodGet, "/realms/acme/api/account/v1/email/verify-context", "", false},
	} {
		got, allowed := requiredAccountScope(tc.method, tc.path)
		if got != tc.scope || allowed != tc.allowed {
			t.Errorf("%s %s = %q,%v; want %q,%v", tc.method, tc.path, got, allowed, tc.scope, tc.allowed)
		}
	}
}

func TestAccountContextAcceptsBothPortalScopes(t *testing.T) {
	for _, tc := range []struct {
		name, path, scope string
		allowed           bool
	}{
		{name: "admin portal", path: "/api/auth/account", scope: "openid profile idmagic.admin", allowed: true},
		{name: "account portal", path: "/api/auth/account", scope: "openid profile idmagic.account", allowed: true},
		{name: "account API client", path: "/api/auth/account", scope: "account:read", allowed: true},
		{name: "unrelated scope", path: "/api/auth/account", scope: "openid profile"},
		{name: "admin scope remains rejected from account API", path: "/api/account/v1/profile", scope: "idmagic.admin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
			req.Header.Set("Authorization", "Bearer jwt")
			req = withAuthTestRealm(req)
			c := e.NewContext(req, httptest.NewRecorder())
			a := Authenticator{AccessTokens: authTestAccessTokens{
				token: &AccessToken{Subject: "user-1", Scope: tc.scope, Audience: []string{authTestRealmAPI}},
			}}

			got, err := a.resolveAuthentication(c)
			if tc.allowed {
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.Subject() != "user-1" {
					t.Fatalf("authn=%+v", got)
				}
				return
			}
			if _, ok := errors.AsType[*InsufficientScopeError](err); !ok {
				t.Fatalf("err=%v; want InsufficientScopeError", err)
			}
		})
	}
}

func TestManagedAccountTokenRequiresActiveRecordAndRouteScope(t *testing.T) {
	base := &AccessToken{Managed: true, Subject: "user-1", ClientID: apitokendomain.BuiltinClientID, Scope: "account:read", Audience: []string{authTestRealmAPI}}
	for _, tc := range []struct {
		name, method, path string
		principal          ApiTokenPrincipal
		authenticated      bool
	}{
		{name: "read", method: http.MethodGet, path: "/api/account/v1/profile", principal: ApiTokenPrincipal{UserID: "user-1", ClientID: apitokendomain.BuiltinClientID}, authenticated: true},
		{name: "write lacks scope", method: http.MethodPatch, path: "/api/account/v1/profile", principal: ApiTokenPrincipal{UserID: "user-1", ClientID: apitokendomain.BuiltinClientID}},
		{name: "missing lifecycle record", method: http.MethodGet, path: "/api/account/v1/profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			req.Header.Set("Authorization", "Bearer jwt")
			req = withAuthTestRealm(req)
			c := e.NewContext(req, httptest.NewRecorder())
			a := Authenticator{AccessTokens: authTestAccessTokens{token: base}, ApiTokens: authTestApiTokens{principal: tc.principal}}
			got, err := a.resolveAuthentication(c)
			if tc.name == "write lacks scope" {
				var scopeErr *InsufficientScopeError
				if !errors.As(err, &scopeErr) || scopeErr.Required != "account:write" {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if tc.name == "missing lifecycle record" {
				if _, ok := errors.AsType[*InvalidTokenError](err); !ok {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if (got != nil) != tc.authenticated {
				t.Fatalf("authn=%+v want=%v", got, tc.authenticated)
			}
		})
	}
}

// account スコープのトークンは、スコープに加えてリクエスト先レルムの IdMagic API を
// audience に持たなければ受けない。ポータル境界のスコープだけのトークンは検査の外に置く。
// 期待する値はリクエストのテナント文脈から取るので、文脈を持たない要求も通さない。
//
//spec:covers REQ-OAUTH2-001, EX-OAUTH2-001-04: account スコープのトークンは aud がリクエスト先レルムの発行者識別子を含むときだけ受理し、client_id、別レルム、audience なし、レルム未解決では InvalidTokenError を返し、ポータル境界のスコープだけのトークンは検査しない。
func TestAccountScopedTokenMustNameTheRealmApi(t *testing.T) {
	for _, tc := range []struct {
		name, scope string
		aud         []string
		inRealm     bool
		accepted    bool
	}{
		{name: "レルムの IdMagic API", scope: "account:read", aud: []string{authTestRealmAPI}, inRealm: true, accepted: true},
		{name: "複数の audience の 1 つ", scope: "account:read", aud: []string{"web-app", authTestRealmAPI}, inRealm: true, accepted: true},
		{name: "client_id", scope: "account:read", aud: []string{"web-app"}, inRealm: true},
		{name: "別レルムの IdMagic API", scope: "account:read", aud: []string{"https://idp.test/realms/other"}, inRealm: true},
		{name: "audience なし", scope: "account:read", inRealm: true},
		{name: "リクエスト先レルムを解決できない", scope: "account:read", aud: []string{authTestRealmAPI}},
		{name: "ポータル境界のスコープだけ", scope: "openid idmagic.account", aud: []string{"web-app"}, inRealm: true, accepted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/account/v1/profile", http.NoBody)
			req.Header.Set("Authorization", "Bearer jwt")
			if tc.inRealm {
				req = withAuthTestRealm(req)
			}
			c := echo.New().NewContext(req, httptest.NewRecorder())
			a := Authenticator{AccessTokens: authTestAccessTokens{
				token: &AccessToken{Subject: "user-1", Scope: tc.scope, Audience: tc.aud},
			}}

			got, err := a.resolveAuthentication(c)
			if tc.accepted {
				if err != nil || got == nil || got.Subject() != "user-1" {
					t.Fatalf("authn=%+v err=%v; want accepted", got, err)
				}
				return
			}
			if _, ok := errors.AsType[*InvalidTokenError](err); !ok {
				t.Fatalf("authn=%+v err=%v; want InvalidTokenError", got, err)
			}
		})
	}
}
