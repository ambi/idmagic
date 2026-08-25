package protocol_oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"maps"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
)

// FuzzVerifyUpstreamIDToken は、こちらの JWKS で署名されていない ID Token が受理されないことを
// 表明する。
//
// 上流 IdP の ID Token は認証の主張そのものであり、ここが緩むと上流を名乗る任意の相手が
// 任意のローカルユーザーとして認証を通せる。alg 混同 (none / HS256)、kid の不一致、
// issuer と audience の不一致、nonce の欠落を corpus に入れて探索させる。
//
// この target は「必ず拒否する」しか言わないので、正当な ID Token が受理されることは
// TestVerifyUpstreamIDTokenAcceptsAValidToken が別に押さえている。
func FuzzVerifyUpstreamIDToken(f *testing.F) {
	connection, attempt, _, keySet, now := fuzzIDTokenFixture(f)

	claims := map[string]any{
		"iss": connection.Issuer, "aud": connection.ClientID, "sub": "upstream-user",
		"nonce": attempt.Nonce, "exp": now.Add(time.Minute).Unix(), "iat": now.Unix(),
	}
	f.Add(unsignedIDToken(f, map[string]any{"alg": "none", "kid": "upstream-key"}, claims, nil))
	f.Add(unsignedIDToken(f, map[string]any{"alg": "HS256", "kid": "upstream-key"}, claims, []byte("mac")))
	f.Add(unsignedIDToken(f, map[string]any{"alg": "RS256", "kid": "unknown"}, claims, []byte("sig")))
	f.Add(unsignedIDToken(f, map[string]any{"alg": "RS256", "kid": "upstream-key"}, claims, []byte("sig")))
	f.Add("")
	f.Add("a.b.c")

	f.Fuzz(func(t *testing.T, idToken string) {
		if len(idToken) > 64*1024 {
			return
		}
		got, err := verifyUpstreamIDToken(idToken, keySet, connection, attempt, now)
		if err == nil {
			t.Fatalf("accepted an ID token the test never signed: %q (claims=%v)", idToken, got)
		}
		if got != nil {
			t.Fatalf("verifyUpstreamIDToken returned %v together with an error", got)
		}
	})
}

// TestVerifyUpstreamIDTokenAcceptsAValidToken は fuzz target が空虚でないことを保つ。
// 正当に署名した ID Token は受理され、nonce・issuer・audience のどれを外しても拒否される。
func TestVerifyUpstreamIDTokenAcceptsAValidToken(t *testing.T) {
	connection, attempt, key, keySet, now := fuzzIDTokenFixture(t)

	base := jwt.MapClaims{
		"iss": connection.Issuer, "aud": connection.ClientID, "sub": "upstream-user",
		"nonce": attempt.Nonce, "exp": now.Add(time.Minute).Unix(), "iat": now.Unix(),
	}
	if _, err := verifyUpstreamIDToken(signIDToken(t, key, base), keySet, connection, attempt, now); err != nil {
		t.Fatalf("a correctly signed ID token was rejected: %v", err)
	}

	for name, mutate := range map[string]func(jwt.MapClaims){
		"wrong issuer":   func(c jwt.MapClaims) { c["iss"] = "https://attacker.example" },
		"wrong audience": func(c jwt.MapClaims) { c["aud"] = "another-client" },
		"missing nonce":  func(c jwt.MapClaims) { delete(c, "nonce") },
		"wrong nonce":    func(c jwt.MapClaims) { c["nonce"] = "other" },
		"expired":        func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Minute).Unix() },
	} {
		t.Run(name, func(t *testing.T) {
			claims := jwt.MapClaims{}
			maps.Copy(claims, base)
			mutate(claims)
			if _, err := verifyUpstreamIDToken(signIDToken(t, key, claims), keySet, connection, attempt, now); err == nil {
				t.Fatalf("accepted an ID token with %s", name)
			}
		})
	}
}

func fuzzIDTokenFixture(tb testing.TB) (
	domain.IdentityProviderConnection,
	domain.FederatedLoginAttempt,
	*rsa.PrivateKey,
	map[string]*rsa.PublicKey,
	time.Time,
) {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatalf("generate key: %v", err)
	}
	connection := domain.IdentityProviderConnection{
		ID: "provider", TenantID: "tenant", Protocol: domain.ProtocolOIDC,
		Status: domain.ConnectionActive,
		Issuer: "https://upstream.example", ClientID: "idmagic-client",
	}
	attempt := domain.FederatedLoginAttempt{
		State: "state", TenantID: "tenant", ProviderID: "provider",
		Protocol: domain.ProtocolOIDC, Nonce: "nonce-1",
	}
	keySet := map[string]*rsa.PublicKey{"upstream-key": &key.PublicKey}
	return connection, attempt, key, keySet, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func signIDToken(tb testing.TB, key *rsa.PrivateKey, claims jwt.MapClaims) string {
	tb.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "upstream-key"
	signed, err := token.SignedString(key)
	if err != nil {
		tb.Fatalf("sign ID token: %v", err)
	}
	return signed
}

func unsignedIDToken(tb testing.TB, header, claims map[string]any, signature []byte) string {
	tb.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		tb.Fatalf("marshal header: %v", err)
	}
	cb, err := json.Marshal(claims)
	if err != nil {
		tb.Fatalf("marshal claims: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(cb) + "." +
		base64.RawURLEncoding.EncodeToString(signature)
}
