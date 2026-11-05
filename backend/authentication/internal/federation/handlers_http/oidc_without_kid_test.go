package handlers_http_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	httpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	federationmemory "github.com/ambi/idmagic/backend/authentication/internal/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	federationhttp "github.com/ambi/idmagic/backend/authentication/internal/federation/handlers_http"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/internal/federation/protocol_oidc"
	federationusecases "github.com/ambi/idmagic/backend/authentication/internal/federation/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/internal/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-AUTHENTICATION-001, EX-AUTHENTICATION-001-04, OIDC-CORE-ID-TOKEN-KID-OPTIONAL: kid を省いた上流の ID Token と暗号化用の鍵が混在する JWKS を実物の OIDCDriver で検証し、既存 User の federated セッションを発行する。
func TestOIDCCallbackWithoutKIDIssuesFederatedSession(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	repos := federationmemory.NewRepositories()
	users := idmanagement.NewMemoryModule().UserRepo
	if err := users.Save(ctx, &userdomain.User{
		ID: "user-alice", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "alice@example.com",
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	connection := &federationdomain.IdentityProviderConnection{
		ID: "provider", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Workforce", Protocol: federationdomain.ProtocolOIDC,
		Status: federationdomain.ConnectionActive, Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth", TokenEndpoint: "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping: federationdomain.ClaimMapping{Subject: "sub", Username: "email"}, LinkingPolicy: federationdomain.LinkingNone,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(ctx, connection); err != nil {
		t.Fatal(err)
	}
	if err := repos.Identities.Create(ctx, &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID,
		ExternalSubject: "external-user", LocalUserID: "user-alice", LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	var authorization url.Values
	var requests []string
	client := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.Method+" "+r.URL.String())
		var body any
		switch r.URL.String() {
		case "https://idp.example/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Method != http.MethodPost || r.Form.Get("code") != "upstream-code" || r.Form.Get("grant_type") != "authorization_code" ||
				r.Form.Get("client_id") != "client" || r.Form.Get("redirect_uri") != "http://idp.test/api/auth/federation/oidc/callback" || r.Form.Get("code_verifier") == "" {
				t.Fatalf("token request method=%s form=%v", r.Method, r.Form)
			}
			// Core §10.1 が許す kid の省略を、製品のトークン生成補助を使わず再現する。
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
				"iss": "https://idp.example", "aud": "client", "sub": "external-user", "email": "alice@example.com",
				"nonce": authorization.Get("nonce"), "exp": now.Add(time.Minute).Unix(),
			})
			signed, err := token.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			body = map[string]any{"id_token": signed, "token_type": "Bearer"}
		case "https://idp.example/jwks":
			body = map[string]any{"keys": []any{
				map[string]any{
					"kty": "RSA", "kid": "encryption", "alg": "RS256", "use": "enc",
					"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
				},
				map[string]any{
					"kty": "RSA", "kid": "signing", "alg": "RS256", "use": "sig",
					"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
				},
			}}
		default:
			t.Fatalf("unexpected upstream request: %s", r.URL)
		}
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(encoded))), Header: make(http.Header)}, nil
	})}}
	sessions := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	e := echo.New()
	federationhttp.RegisterRoutes(e.Group("", testing_tenant.ResolveDefault), federationhttp.Deps{
		Broker: federationusecases.BrokerDeps{
			Connections: repos.Connections, Identities: repos.Identities, Attempts: repos.Attempts, Users: users, Sessions: sessions,
			Drivers: map[federationdomain.Protocol]federationusecases.ProtocolDriver{
				federationdomain.ProtocolOIDC: federationhttp.OIDCDriver{Client: client},
			},
		},
		Auth: httpdeps.Deps{Deps: support.Deps{Issuer: "http://idp.test"}},
	})
	started := httptest.NewRecorder()
	e.ServeHTTP(started, httptest.NewRequest(http.MethodGet, "/api/auth/federation/start?provider_id=provider", http.NoBody))
	location, err := url.Parse(started.Header().Get("Location"))
	if err != nil || started.Code != http.StatusSeeOther || location.Scheme != "https" || location.Host != "idp.example" || location.Path != "/auth" {
		t.Fatalf("start status=%d location=%v err=%v", started.Code, location, err)
	}
	authorization = location.Query()
	if authorization.Get("state") == "" || authorization.Get("nonce") == "" {
		t.Fatalf("authorization query=%v", authorization)
	}
	completed := httptest.NewRecorder()
	e.ServeHTTP(completed, httptest.NewRequest(http.MethodGet,
		"/api/auth/federation/oidc/callback?state="+url.QueryEscape(authorization.Get("state"))+"&code=upstream-code", http.NoBody))
	if completed.Code != http.StatusSeeOther || completed.Header().Get("Location") != "/authorize/resume" {
		t.Fatalf("callback status=%d headers=%v body=%s", completed.Code, completed.Header(), completed.Body.String())
	}
	cookies := completed.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value == "" {
		t.Fatalf("session cookies=%v", cookies)
	}
	authn, err := sessions.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: http.Header{"Cookie": {cookies[0].String()}}})
	if err != nil || authn == nil || authn.UserID != "user-alice" || !slices.Equal(authn.AMR, []string{authdomain.AMRFederated}) {
		t.Fatalf("authn=%+v err=%v, want user-alice with federated AMR", authn, err)
	}
	if !slices.Equal(requests, []string{"POST https://idp.example/token", "GET https://idp.example/jwks"}) {
		t.Fatalf("upstream requests=%v", requests)
	}
}
