package support_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/labstack/echo/v5"
)

type authTestIntrospector struct {
	result *oauthports.IntrospectionResult
}

func (f authTestIntrospector) IntrospectAccessToken(context.Context, string) (*oauthports.IntrospectionResult, error) {
	return f.result, nil
}

type authTestManagedAuthenticator struct {
	principal apitokendomain.Principal
	err       error
}

func (f authTestManagedAuthenticator) Authenticate(context.Context, string) (apitokendomain.Principal, error) {
	return f.principal, f.err
}

// wi-275: account resource server の route と最小 scope の正準対応。
func TestRequiredAccountScope(t *testing.T) {
	for _, tc := range []struct {
		method, path, scope string
		allowed             bool
	}{
		{http.MethodGet, "/realms/acme/api/account/profile", "account:read", true},
		{http.MethodPatch, "/realms/acme/api/account/profile", "account:write", true},
		{http.MethodPost, "/realms/acme/api/account/mfa/totp/remove", "account:mfa:write", true},
		{http.MethodPost, "/realms/acme/api/account/sessions/s1/revoke", "account:sessions:write", true},
		{http.MethodPost, "/realms/acme/api/account/consents/c1/revoke", "account:consents:write", true},
		{http.MethodPost, "/realms/acme/api/auth/change_password", "account:password:write", true},
		{http.MethodPost, "/realms/acme/api/account/step_up/start", "", false},
		{http.MethodGet, "/realms/acme/api/account/email/verify_context", "", false},
	} {
		got, allowed := requiredAccountScope(tc.method, tc.path)
		if got != tc.scope || allowed != tc.allowed {
			t.Errorf("%s %s = %q,%v; want %q,%v", tc.method, tc.path, got, allowed, tc.scope, tc.allowed)
		}
	}
}

func TestManagedAccountTokenRequiresActiveRecordAndRouteScope(t *testing.T) {
	base := &oauthports.IntrospectionResult{Active: true, Managed: true, Sub: "user-1", ClientID: apitokendomain.BuiltinClientID, Scope: "account:read"}
	for _, tc := range []struct {
		name, method, path string
		principal          apitokendomain.Principal
		authenticated      bool
	}{
		{name: "read", method: http.MethodGet, path: "/api/account/profile", principal: apitokendomain.Principal{UserID: "user-1", ClientID: apitokendomain.BuiltinClientID}, authenticated: true},
		{name: "write lacks scope", method: http.MethodPatch, path: "/api/account/profile", principal: apitokendomain.Principal{UserID: "user-1", ClientID: apitokendomain.BuiltinClientID}},
		{name: "missing lifecycle record", method: http.MethodGet, path: "/api/account/profile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tc.method, tc.path, http.NoBody)
			req.Header.Set("Authorization", "Bearer jwt")
			c := e.NewContext(req, httptest.NewRecorder())
			a := Authenticator{TokenIntrospector: authTestIntrospector{result: base}, ApiTokenAuthenticator: authTestManagedAuthenticator{principal: tc.principal}}
			got, err := a.resolveAuthnContext(c)
			if tc.name == "write lacks scope" {
				var scopeErr *InsufficientScopeError
				if !errors.As(err, &scopeErr) || scopeErr.Required != "account:write" {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if tc.name == "missing lifecycle record" {
				var tokenErr *InvalidTokenError
				if !errors.As(err, &tokenErr) {
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
