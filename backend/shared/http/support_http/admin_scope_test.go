package support_http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// adminScopeContext は router が一致させたルートテンプレートを持つリクエストを組み立てる。
// 契約への解決はテンプレートを鍵にするため、テンプレートを持たない Context では
// 粒度スコープの判定そのものが再現しない。
func adminScopeContext(method, routePath string, scopes ...apitokendomain.Scope) (*echo.Context, Authenticator) {
	e := echo.New()
	request := httptest.NewRequest(method, "/realms/acme"+routePath, http.NoBody)
	request.Header.Set("Authorization", "Bearer jwt")
	c := e.NewContext(request, httptest.NewRecorder())
	c.SetPath("/realms/:tenant_id" + routePath)
	introspection := &oauthports.IntrospectionResult{
		Active: true, Managed: true, Sub: "user-1",
		ClientID: apitokendomain.BuiltinClientID,
		Scope:    strings.Join(apitokendomain.Scopes(scopes).Strings(), " "),
	}
	return c, Authenticator{
		TokenIntrospector: authTestIntrospector{result: introspection},
		ApiTokenAuthenticator: authTestManagedAuthenticator{principal: apitokendomain.Principal{
			UserID: "user-1", ClientID: apitokendomain.BuiltinClientID, Scopes: scopes,
		}},
	}
}

// TestAdminApiTokenScopeEnforcement は REQ-APITOKENS-004 の粒度判定を固定する。
// 対応するスコープを持つ API アクセストークンだけが管理 API へ到達し、別リソースの
// スコープ、同一リソースの参照系スコープ、対話セッション限定の operation はいずれも
// 拒否される。
func TestAdminApiTokenScopeEnforcement(t *testing.T) {
	for _, tc := range []struct {
		name, method, routePath string
		granted                 []apitokendomain.Scope
		wantRequired            string
	}{
		{
			name: "granular scope reaches the admin API", method: http.MethodGet,
			routePath: "/api/admin/v1/saml/service-providers",
			granted:   []apitokendomain.Scope{apitokendomain.ScopeSamlRead},
		},
		{
			name: "read scope does not reach a write operation", method: http.MethodPost,
			routePath: "/api/admin/v1/saml/service-providers",
			granted:   []apitokendomain.Scope{apitokendomain.ScopeSamlRead},
			// REQ-SAML-005: saml:read だけで変更操作を要求すると拒否される。
			wantRequired: "saml:write",
		},
		{
			// REQ-WSFEDERATION-001: wsfed:read だけで変更操作を要求すると拒否される。
			// 参照のスコープが変更へ届かないことは、そのスコープを配った相手が信頼設定を
			// 書き換えられないことそのものなので、宣言だけでなくここで固定する。
			name: "ws-federation read scope does not reach a write operation", method: http.MethodPost,
			routePath:    "/api/admin/v1/wsfed/relying-parties",
			granted:      []apitokendomain.Scope{apitokendomain.ScopeWsFedRead},
			wantRequired: "wsfed:write",
		},
		{
			name: "another resource's scope does not reach", method: http.MethodGet,
			routePath: "/api/admin/v1/users",
			granted:   []apitokendomain.Scope{apitokendomain.ScopeSamlRead},
			// REQ-OAUTH2-003: 別 resource の scope で操作を要求すると拒否される。
			wantRequired: "users:read",
		},
		{
			name: "write scope reaches a write operation", method: http.MethodPost,
			routePath: "/api/admin/v1/agents/:agent_id/kill",
			granted:   []apitokendomain.Scope{apitokendomain.ScopeAgentsWrite},
		},
		{
			name: "interactive-session-only operation rejects every token", method: http.MethodPost,
			routePath:    "/api/admin/v1/api-tokens",
			granted:      []apitokendomain.Scope{apitokendomain.ScopeUsersWrite},
			wantRequired: spec.InteractiveSessionScope,
		},
		{
			name: "operation outside the contract is fail-closed", method: http.MethodGet,
			routePath:    "/api/admin/v1/not-in-the-contract",
			granted:      []apitokendomain.Scope{apitokendomain.ScopeUsersRead},
			wantRequired: spec.InteractiveSessionScope,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, authenticator := adminScopeContext(tc.method, tc.routePath, tc.granted...)
			authn, err := authenticator.resolveAuthnContext(c)
			if tc.wantRequired == "" {
				if err != nil {
					t.Fatalf("err = %v, want the call to be authorized", err)
				}
				if authn == nil || authn.UserID != "user-1" {
					t.Fatalf("authn = %+v, want the token's issuer", authn)
				}
				return
			}
			var scopeErr *InsufficientScopeError
			if !errors.As(err, &scopeErr) {
				t.Fatalf("err = %v, want InsufficientScopeError", err)
			}
			if scopeErr.Required != tc.wantRequired {
				t.Fatalf("required scope = %q, want %q", scopeErr.Required, tc.wantRequired)
			}
		})
	}
}

// TestAdminPortalTokenSkipsGranularScopes は、粒度スコープを持たない主体の経路が
// 変わっていないことを確かめる。ブラウザーのポータルが提示する通常の OAuth アクセス
// トークンは従来どおり idmagic.admin だけで判定する。
func TestAdminPortalTokenSkipsGranularScopes(t *testing.T) {
	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/realms/acme/api/admin/v1/saml/service-providers", http.NoBody)
	request.Header.Set("Authorization", "Bearer jwt")
	c := e.NewContext(request, httptest.NewRecorder())
	c.SetPath("/realms/:tenant_id/api/admin/v1/saml/service-providers")
	authenticator := Authenticator{TokenIntrospector: authTestIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user-1", Scope: "openid idmagic.admin",
	}}}
	authn, err := authenticator.resolveAuthnContext(c)
	if err != nil {
		t.Fatalf("err = %v, want the portal token to remain authorized", err)
	}
	if authn == nil || authn.UserID != "user-1" {
		t.Fatalf("authn = %+v, want the portal user", authn)
	}
}

// TestAdminApiTokenInsufficientScopeChallenge は RFC6750-API-TOKEN-ERROR を固定する。
// スコープ不足は insufficient_scope と必要なスコープ名を返し、ロール不足の access_denied
// とは区別できる。
func TestAdminApiTokenInsufficientScopeChallenge(t *testing.T) {
	e := echo.New()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/v1/saml/service-providers", http.NoBody)
	recorder := httptest.NewRecorder()
	c := e.NewContext(request, recorder)

	handled, err := WriteAccessTokenError(c, &InsufficientScopeError{Required: "saml:write"})
	if err != nil || !handled {
		t.Fatalf("handled=%v err=%v, want the scope failure to be handled", handled, err)
	}
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if got := recorder.Header().Get("WWW-Authenticate"); got != `Bearer error="insufficient_scope", scope="saml:write"` {
		t.Fatalf("WWW-Authenticate = %q", got)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "insufficient_scope") {
		t.Fatalf("body = %q, want the insufficient_scope problem type", body)
	}
}
