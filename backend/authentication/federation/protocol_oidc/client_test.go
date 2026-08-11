package protocol_oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
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
