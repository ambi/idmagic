package support_http

import (
	"context"
	cryptostd "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
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

// authTestDPoPProof signs a DPoP proof JWT with the given htm / htu / ath.
func authTestDPoPProof(t *testing.T, key *rsa.PrivateKey, jwk map[string]any, htm, htu, jti, ath string, now time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"htm": htm, "htu": htu, "jti": jti, "iat": now.Unix()}
	if ath != "" {
		claims["ath"] = ath
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPSS(rand.Reader, key, cryptostd.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func authTestJWK(pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(pub.E)).Bytes()),
	}
}

func authTestJKT(t *testing.T, jwk map[string]any) string {
	t.Helper()
	canonical, err := json.Marshal(map[string]any{"e": jwk["e"], "kty": jwk["kty"], "n": jwk["n"]})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// REQ-OAUTH2-045: a DPoP proof at a protected resource is bound to the presented access
// token through ath. A proof that only demonstrates key possession (no ath), or one made
// for another token, is rejected.
func TestResourceDPoPProofBindsToPresentedAccessToken(t *testing.T) {
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := authTestJWK(&key.PublicKey)
	const (
		path        = "/api/account/v1/profile"
		accessToken = "AT1"
		otherToken  = "AT2"
	)

	for _, tc := range []struct {
		name          string
		ath           string
		authenticated bool
	}{
		{name: "ath of the presented token", ath: tokensjose.AccessTokenHash(accessToken), authenticated: true},
		{name: "no ath", ath: ""},
		{name: "ath of another access token", ath: tokensjose.AccessTokenHash(otherToken)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			req.Header.Set("Authorization", "DPoP "+accessToken)
			req.Header.Set("DPoP", authTestDPoPProof(t, key, jwk, http.MethodGet, path, tc.name, tc.ath, now))
			c := e.NewContext(req, httptest.NewRecorder())
			a := Authenticator{
				TokenIntrospector: authTestIntrospector{result: &oauthports.IntrospectionResult{
					Active: true, Sub: "user-1", Scope: "account:read",
					SenderConstraint: &oauthdomain.SenderConstraint{
						Type: spec.SenderConstraintDPoP, JKT: authTestJKT(t, jwk),
					},
				}},
				DpopReplayStore: memory.NewDpopReplayStore(),
			}

			got, err := a.resolveAuthnContext(c)
			if tc.authenticated {
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.UserID != "user-1" {
					t.Fatalf("authn=%+v", got)
				}
				return
			}
			var tokenErr *InvalidTokenError
			if !errors.As(err, &tokenErr) {
				t.Fatalf("err=%v authn=%+v; want InvalidTokenError", err, got)
			}
		})
	}
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
		{http.MethodPost, "/realms/acme/api/auth/change_password", "account:password:write", true},
		{http.MethodPost, "/realms/acme/api/account/v1/step_up/start", "", false},
		{http.MethodGet, "/realms/acme/api/account/v1/email/verify_context", "", false},
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
			c := e.NewContext(req, httptest.NewRecorder())
			a := Authenticator{TokenIntrospector: authTestIntrospector{
				result: &oauthports.IntrospectionResult{Active: true, Sub: "user-1", Scope: tc.scope},
			}}

			got, err := a.resolveAuthnContext(c)
			if tc.allowed {
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.UserID != "user-1" {
					t.Fatalf("authn=%+v", got)
				}
				return
			}
			var scopeErr *InsufficientScopeError
			if !errors.As(err, &scopeErr) {
				t.Fatalf("err=%v; want InsufficientScopeError", err)
			}
		})
	}
}

func TestManagedAccountTokenRequiresActiveRecordAndRouteScope(t *testing.T) {
	base := &oauthports.IntrospectionResult{Active: true, Managed: true, Sub: "user-1", ClientID: apitokendomain.BuiltinClientID, Scope: "account:read"}
	for _, tc := range []struct {
		name, method, path string
		principal          apitokendomain.Principal
		authenticated      bool
	}{
		{name: "read", method: http.MethodGet, path: "/api/account/v1/profile", principal: apitokendomain.Principal{UserID: "user-1", ClientID: apitokendomain.BuiltinClientID}, authenticated: true},
		{name: "write lacks scope", method: http.MethodPatch, path: "/api/account/v1/profile", principal: apitokendomain.Principal{UserID: "user-1", ClientID: apitokendomain.BuiltinClientID}},
		{name: "missing lifecycle record", method: http.MethodGet, path: "/api/account/v1/profile"},
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
