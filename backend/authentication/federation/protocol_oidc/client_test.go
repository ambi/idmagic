package protocol_oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRefreshDiscoveryRejectsIssuerMixup(t *testing.T) {
	client := Client{HTTPClient: fakeHTTPClient(map[string]any{
		"https://idp.example/.well-known/openid-configuration": map[string]any{
			"issuer":                 "https://attacker.example",
			"authorization_endpoint": "https://idp.example/auth",
			"token_endpoint":         "https://idp.example/token",
			"jwks_uri":               "https://idp.example/jwks",
		},
	})}
	connection := testConnection()
	if err := client.RefreshDiscovery(context.Background(), &connection, time.Now()); err == nil {
		t.Fatal("issuer mismatch must be rejected")
	}
}

func TestAuthorizationURLUsesStateNonceAndPKCES256(t *testing.T) {
	connection := testConnection()
	attempt := domain.FederatedLoginAttempt{State: "state", Nonce: "nonce", PKCEVerifier: strings.Repeat("v", 43)}
	redirect, err := AuthorizationURL(connection, attempt, "https://broker.example/callback")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"state=state", "nonce=nonce", "code_challenge_method=S256", "response_type=code"} {
		if !strings.Contains(redirect, part) {
			t.Fatalf("authorization URL %q missing %q", redirect, part)
		}
	}
}

func TestExchangeValidatesSignedIDTokenAndNormalizesClaims(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": "https://idp.example", "aud": "client", "sub": "subject",
		"email": "user@example.com", "email_verified": true, "name": "User",
		"nonce": "nonce", "iat": now.Unix(), "exp": now.Add(5 * time.Minute).Unix(),
	})
	token.Header["kid"] = "key-1"
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "key-1", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}}
	client := Client{HTTPClient: fakeHTTPClient(map[string]any{
		"https://idp.example/token": map[string]any{"id_token": signed, "token_type": "Bearer"},
		"https://idp.example/jwks":  jwks,
	})}
	connection := testConnection()
	claims, err := client.ExchangeAndValidate(
		context.Background(), connection,
		domain.FederatedLoginAttempt{Nonce: "nonce", PKCEVerifier: strings.Repeat("v", 43)},
		"code", "https://broker.example/callback", now,
	)
	if err != nil {
		t.Fatalf("ExchangeAndValidate: %v", err)
	}
	if claims.Subject != "subject" || claims.Username != "user@example.com" || !claims.EmailVerified {
		t.Fatalf("claims=%+v", claims)
	}
}

// RED (interface: TestIdentityProviderConnection): reachable endpoints and a
// resolvable secret_reference together report success with no failures.
// RED (bug found via manual verification): after envelope encryption, the repository
// hands protocol_oidc the already-decrypted plaintext secret directly (or, in the memory
// backend, whatever value was saved), not an "env:" reference. That value must be usable
// as-is without a SecretResolver — only a literal "env:" prefix should require resolution.
func TestTestConnectionAcceptsAlreadyResolvedSecretWithoutAResolver(t *testing.T) {
	client := Client{
		HTTPClient: fakeHTTPClient(map[string]any{
			"https://idp.example/auth":  map[string]any{},
			"https://idp.example/token": map[string]any{},
			"https://idp.example/jwks":  map[string]any{},
		}),
	}
	connection := testConnection()
	connection.SecretReference = "s3cr3t-plaintext-client-secret"
	if failures := client.TestConnection(context.Background(), connection); len(failures) != 0 {
		t.Fatalf("failures=%v, want none (SecretResolver is nil but the value is not an env: reference)", failures)
	}
}

func TestTestConnectionReportsSuccessWhenEndpointsReachableAndSecretResolves(t *testing.T) {
	client := Client{
		HTTPClient: fakeHTTPClient(map[string]any{
			"https://idp.example/auth":  map[string]any{},
			"https://idp.example/token": map[string]any{},
			"https://idp.example/jwks":  map[string]any{},
		}),
		SecretResolver: stubResolver{value: "shh"},
	}
	connection := testConnection()
	connection.SecretReference = "env:CLIENT_SECRET"
	if failures := client.TestConnection(context.Background(), connection); len(failures) != 0 {
		t.Fatalf("failures=%v, want none", failures)
	}
}

// RED: an unreachable JWKS endpoint and an unresolvable secret_reference are both reported,
// without leaking the secret value itself.
func TestTestConnectionReportsUnreachableEndpointAndUnresolvableSecret(t *testing.T) {
	client := Client{
		HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == "https://idp.example/jwks" {
				return nil, errors.New("connection refused")
			}
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: make(http.Header)}, nil
		})},
		SecretResolver: stubResolver{err: errors.New("referenced environment secret is unavailable")},
	}
	connection := testConnection()
	connection.SecretReference = "env:MISSING"
	failures := client.TestConnection(context.Background(), connection)
	if len(failures) != 2 {
		t.Fatalf("failures=%v, want 2 (jwks unreachable + secret unresolved)", failures)
	}
	for _, failure := range failures {
		if strings.Contains(failure, "referenced environment secret is unavailable") {
			t.Fatalf("failure message leaked resolver error detail: %q", failure)
		}
	}
}

type stubResolver struct {
	value string
	err   error
}

func (s stubResolver) Resolve(context.Context, string) (string, error) { return s.value, s.err }

func testConnection() domain.IdentityProviderConnection {
	return domain.IdentityProviderConnection{
		ID: "oidc", TenantID: "tenant", DisplayName: "OIDC",
		Protocol: domain.ProtocolOIDC, Status: domain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping: domain.ClaimMapping{
			Subject: "sub", Username: "email", Email: "email",
			EmailVerified: "email_verified", Name: "name",
		},
		LinkingPolicy: domain.LinkingNone,
	}
}

func fakeHTTPClient(responses map[string]any) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		value, ok := responses[request.URL.String()]
		if !ok {
			return &http.Response{
				StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("{}")),
				Header: http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		}
		body, _ := json.Marshal(value)
		return &http.Response{
			StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body))),
			Header: http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}
}

// 言っているので観測も 2 つ要る。issuer が「設定した発行者と完全一致」であること —
// 前後に何かが付いた値や末尾のスラッシュ違いは別の発行者である — と、endpoint と JWKS URI
// が HTTPS の公開オーソリティに限られることである。どの拒否でも connection の
// endpoint が書き換わっていないことを併せて観測する。error を返してから endpoint を
// 更新する実装は、戻り値だけを見ると正しい実装と区別が付かない。
//
// 拒否の観測だけを並べると、別の理由 (fixture の JWKS が引けない、など) ですべて落ちて
// いるだけの状態を「検証している」と読み違える。差し替える要素以外はすべて正当な
// document を fixture にし、無傷の document が受理されて connection を書き換えることを
// 対照として最後に観測する。
//
//spec:covers OIDC-DISCOVERY-ISSUER: Discovery Metadata の受け入れ条件を固定する。行は 2 つのことを
func TestRefreshDiscoveryRequiresAnExactIssuerAndHTTPSAuthorities(t *testing.T) {
	const configured = "https://idp.example"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "key-1", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
	}}}
	// 差し替えを受けていない document。どの endpoint も引けるので、拒否が起きたなら
	// 理由は差し替えた要素しかない。
	intact := func() map[string]any {
		return map[string]any{
			"issuer":                 configured,
			"authorization_endpoint": "https://idp.example/auth2",
			"token_endpoint":         "https://idp.example/token2",
			"jwks_uri":               "https://idp.example/jwks2",
		}
	}
	refresh := func(t *testing.T, document map[string]any) (domain.IdentityProviderConnection, error) {
		t.Helper()
		client := Client{HTTPClient: fakeHTTPClient(map[string]any{
			configured + "/.well-known/openid-configuration": document,
			"https://idp.example/jwks2":                      jwks,
			"https://idp.example/jwks":                       jwks,
			"http://idp.example/jwks2":                       jwks,
			"https://127.0.0.1/jwks2":                        jwks,
			"https://10.0.0.1/jwks":                          jwks,
			"https://user:pass@idp.example/jwks":             jwks,
		})}
		connection := testConnection()
		err := client.RefreshDiscovery(context.Background(), &connection, time.Now())
		return connection, err
	}

	for name, forged := range map[string]map[string]any{
		// 完全一致でない issuer。いずれも「前方一致」「後方一致」「末尾スラッシュを無視」
		// のどれかを許す実装なら通ってしまう。
		"issuer with a trailing slash": {"issuer": configured + "/"},
		"issuer with a suffix":         {"issuer": configured + ".evil.test"},
		"issuer with a prefix":         {"issuer": "https://evil.test/" + configured},
		"issuer of another provider":   {"issuer": "https://attacker.example"},
		// HTTPS の公開オーソリティに限らない endpoint。
		"plaintext authorization endpoint": {"authorization_endpoint": "http://idp.example/auth"},
		"plaintext JWKS URI":               {"jwks_uri": "http://idp.example/jwks2"},
		"loopback token endpoint":          {"token_endpoint": "https://127.0.0.1/token"},
		"private network JWKS URI":         {"jwks_uri": "https://10.0.0.1/jwks"},
		"JWKS URI carrying userinfo":       {"jwks_uri": "https://user:pass@idp.example/jwks"},
	} {
		document := intact()
		maps.Copy(document, forged)
		connection, err := refresh(t, document)
		if err == nil {
			t.Fatalf("%s: accepted", name)
		}
		before := testConnection()
		if connection.AuthorizationEndpoint != before.AuthorizationEndpoint ||
			connection.TokenEndpoint != before.TokenEndpoint ||
			connection.JWKSURI != before.JWKSURI ||
			connection.MetadataRefreshedAt != nil {
			t.Fatalf("%s: the rejected document still moved the connection to %+v", name, connection)
		}
	}

	// 対照: 無傷の document は受理され、connection の endpoint を書き換える。
	connection, err := refresh(t, intact())
	if err != nil {
		t.Fatalf("an intact discovery document was rejected: %v", err)
	}
	if connection.AuthorizationEndpoint != "https://idp.example/auth2" ||
		connection.TokenEndpoint != "https://idp.example/token2" ||
		connection.JWKSURI != "https://idp.example/jwks2" ||
		connection.MetadataRefreshedAt == nil {
		t.Fatalf("the accepted document did not move the connection: %+v", connection)
	}
}
