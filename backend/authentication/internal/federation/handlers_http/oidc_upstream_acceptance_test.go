package handlers_http_test

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	"github.com/labstack/echo/v5"
)

// upstreamIdP は、上流の OIDC IdP のトークンエンドポイントと JWK Set の応答を決める。
// tokenResponse は、callback の試行が発行した nonce を受け取り、トークン応答の本文を返す。
type upstreamIdP struct {
	tokenResponse func(t *testing.T, nonce string) map[string]any
	jwks          map[string]any
}

// federatedCallback は、連携済みの既存 User を持つ OIDC 接続で start から callback までを一度
// 通し、callback の応答と、応答の cookie から解決したセッションを返す。組み立ては
// TestOIDCCallbackWithoutKIDIssuesFederatedSession と同じで、上流の応答だけを差し替える。
func federatedCallback(t *testing.T, upstream upstreamIdP) (*httptest.ResponseRecorder, *authdomain.AuthenticationContext) {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	now := time.Now().UTC()
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
	client := oidcprotocol.Client{HTTPClient: &http.Client{Transport: fakeRoundTripper(func(r *http.Request) (*http.Response, error) {
		var body any
		switch r.URL.String() {
		case "https://idp.example/token":
			body = upstream.tokenResponse(t, authorization.Get("nonce"))
		case "https://idp.example/jwks":
			body = upstream.jwks
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
	if err != nil || started.Code != http.StatusSeeOther {
		t.Fatalf("start status=%d location=%v err=%v", started.Code, location, err)
	}
	authorization = location.Query()
	completed := httptest.NewRecorder()
	e.ServeHTTP(completed, httptest.NewRequest(http.MethodGet,
		"/api/auth/federation/oidc/callback?state="+url.QueryEscape(authorization.Get("state"))+"&code=upstream-code", http.NoBody))
	var authn *authdomain.AuthenticationContext
	for _, cookie := range completed.Result().Cookies() {
		if cookie.Value == "" {
			continue
		}
		resolved, err := sessions.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: http.Header{"Cookie": {cookie.String()}}})
		if err == nil && resolved != nil {
			authn = resolved
		}
	}
	return completed, authn
}

// upstreamClaims は、接続と試行に一致する正しい ID Token のクレームを返す。
func upstreamClaims(nonce string) map[string]any {
	return map[string]any{
		"iss": "https://idp.example", "aud": "client", "sub": "external-user", "email": "alice@example.com",
		"nonce": nonce, "exp": time.Now().Add(time.Minute).Unix(),
	}
}

func rsaJWK(key *rsa.PrivateKey, kid string) map[string]any {
	return map[string]any{
		"kty": "RSA", "kid": kid, "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}
}

// signJWS は header と claims から JWS を作る。署名は sign に任せ、製品の署名補助を通さない。
func signJWS(t *testing.T, header, claims map[string]any, sign func(digest []byte) []byte) string {
	t.Helper()
	encodedHeader, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(encodedHeader) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sign(digest[:]))
}

// OpenID Connect Core §3.1.3.3 は、トークン応答の形を RFC 6749 §5.1 に委ねる。製品は応答から
// `id_token` だけを読むので、`token_type`、`expires_in`、`refresh_token` を省いた応答でも
// サインインを完了する。
//
// 完了は、callback が federated セッションを発行し、そのセッションが連携済みの User に
// 解決されることで読む。省いた要素を読もうとする実装は、ここでセッションを発行しない。
//
//spec:covers OIDC-CORE-TOKEN-RESPONSE-OPTIONAL: token_type、expires_in、refresh_token を省き id_token だけを返す上流で、callback が連携済みの User の federated セッションを発行する。
func TestOIDCCallbackAcceptsATokenResponseWithOnlyAnIDToken(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	completed, authn := federatedCallback(t, upstreamIdP{
		tokenResponse: func(t *testing.T, nonce string) map[string]any {
			t.Helper()
			signed := signJWS(t, map[string]any{"alg": "RS256", "kid": "signing", "typ": "JWT"}, upstreamClaims(nonce),
				func(digest []byte) []byte {
					signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest)
					if err != nil {
						t.Fatal(err)
					}
					return signature
				})
			return map[string]any{"id_token": signed}
		},
		jwks: map[string]any{"keys": []any{rsaJWK(key, "signing")}},
	})
	if completed.Code != http.StatusSeeOther || completed.Header().Get("Location") != "/authorize/resume" {
		t.Fatalf("callback status=%d headers=%v body=%s", completed.Code, completed.Header(), completed.Body.String())
	}
	if authn == nil || authn.UserID != "user-alice" {
		t.Fatalf("authn=%+v, want the linked user-alice", authn)
	}
}

// OpenID Connect Core §3.1.3.7 は、ID Token の `alg` を RS256 か登録時に示したアルゴリズムと
// することを SHOULD とする。製品は RS256 だけを検証するので、ES256 で署名された ID Token を
// 拒否する。
//
// JWK Set には、ID Token を署名した EC 鍵と同じ `kid` で、RS256 の鍵も置く。`alg` に従って
// 検証の方式を選ぶ実装は EC 鍵で検証を通し、セッションを発行する。
//
//spec:covers OIDC-CORE-ID-TOKEN-RS256 (partial): ES256 で署名した上流の ID Token では、callback がサインインを完了せずセッションを発行しない。
func TestOIDCCallbackRejectsAnIDTokenSignedWithES256(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecJWK := map[string]any{
		"kty": "EC", "crv": "P-256", "kid": "signing", "alg": "ES256", "use": "sig",
		"x": base64.RawURLEncoding.EncodeToString(ecKey.X.FillBytes(make([]byte, 32))),
		"y": base64.RawURLEncoding.EncodeToString(ecKey.Y.FillBytes(make([]byte, 32))),
	}
	completed, authn := federatedCallback(t, upstreamIdP{
		tokenResponse: func(t *testing.T, nonce string) map[string]any {
			t.Helper()
			signed := signJWS(t, map[string]any{"alg": "ES256", "kid": "signing", "typ": "JWT"}, upstreamClaims(nonce),
				func(digest []byte) []byte {
					r, s, err := ecdsa.Sign(rand.Reader, ecKey, digest)
					if err != nil {
						t.Fatal(err)
					}
					return append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...)
				})
			return map[string]any{"id_token": signed, "token_type": "Bearer"}
		},
		jwks: map[string]any{"keys": []any{ecJWK, rsaJWK(rsaKey, "signing")}},
	})
	if completed.Header().Get("Location") == "/authorize/resume" {
		t.Fatalf("ES256 の ID Token でサインインが完了した: status=%d headers=%v", completed.Code, completed.Header())
	}
	if authn != nil {
		t.Fatalf("ES256 の ID Token でセッションが発行された: %+v", authn)
	}
}
