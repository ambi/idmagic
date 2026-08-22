package tokens_jose

import (
	"context"
	cryptostd "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
)

func encodeClientAssertion(t *testing.T, key *rsa.PrivateKey, header, payload map[string]any) string {
	t.Helper()
	hb, err := json.Marshal(header)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(hb) + "." +
		base64.RawURLEncoding.EncodeToString(pb)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPSS(
		rand.Reader, key, cryptostd.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	)
	if err != nil {
		t.Fatal(err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyAudience(t *testing.T) {
	expected := []string{"https://idp.example/token", "https://idp.example/par"}
	cases := []struct {
		name string
		aud  any
		want bool
	}{
		{"matching string", "https://idp.example/token", true},
		{"non-matching string", "https://other.example", false},
		{"matching entry in array", []any{"https://other.example", "https://idp.example/par"}, true},
		{"no matching entry in array", []any{"https://other.example"}, false},
		{"array with non-string entries", []any{42, true}, false},
		{"unsupported type", 42, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := verifyAudience(tc.aud, expected); got != tc.want {
				t.Fatalf("verifyAudience(%#v) = %v, want %v", tc.aud, got, tc.want)
			}
		})
	}
}

func TestPickJWK(t *testing.T) {
	_, jwkA := dpopTestKey(t)
	jwkA["kid"] = "key-a"
	_, jwkB := dpopTestKey(t)
	jwkB["kid"] = "key-b"

	t.Run("no kid with a single key resolves it", func(t *testing.T) {
		if _, err := pickJWK([]map[string]any{jwkA}, ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("no kid with multiple keys is an error", func(t *testing.T) {
		if _, err := pickJWK([]map[string]any{jwkA, jwkB}, ""); err == nil {
			t.Fatal("expected error requiring kid")
		}
	})

	t.Run("kid selects the matching key", func(t *testing.T) {
		if _, err := pickJWK([]map[string]any{jwkA, jwkB}, "key-b"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unknown kid is an error", func(t *testing.T) {
		if _, err := pickJWK([]map[string]any{jwkA}, "missing"); err == nil {
			t.Fatal("expected error for unknown kid")
		}
	})

	t.Run("propagates a malformed jwk error", func(t *testing.T) {
		bad := map[string]any{"kid": "key-a", "kty": "RSA", "n": "not-base64!", "e": jwkA["e"]}
		if _, err := pickJWK([]map[string]any{bad}, "key-a"); err == nil {
			t.Fatal("expected error for malformed jwk")
		}
	})
}

func TestVerifyClientAssertionRejectsMalformedInput(t *testing.T) {
	const (
		clientID = "private-client"
		audience = "https://idp.example/token"
	)
	key, jwk := dpopTestKey(t)
	jwk["kid"] = "client-key"
	replay := memory.NewClientAssertionReplayStore()
	keysFn := func(_ context.Context, _ string) ([]map[string]any, error) { //nolint:unparam // error is always nil here; these cases all fail before the jwks load
		return []map[string]any{jwk}, nil
	}
	now := time.Now().UTC()

	base := func() (map[string]any, map[string]any) {
		return map[string]any{"alg": "PS256", "kid": "client-key"},
			map[string]any{
				"iss": clientID, "sub": clientID, "aud": audience, "jti": "jti-x",
				"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
			}
	}

	cases := []struct {
		name      string
		mutate    func(h, p map[string]any)
		wantError string
	}{
		{"malformed jwt", nil, "malformed JWT"},
		{"disallowed alg", func(h, _ map[string]any) { h["alg"] = "HS256" }, "not allowed"},
		{"sub does not match iss", func(_, p map[string]any) { p["sub"] = "someone-else" }, "iss must equal sub"},
		{"iss does not match client_id", func(_, p map[string]any) { p["iss"] = "other-client"; p["sub"] = "other-client" }, "does not match client_id"},
		{"missing exp", func(_, p map[string]any) { delete(p, "exp") }, "exp required"},
		{"lifetime too long", func(_, p map[string]any) { p["exp"] = now.Add(time.Hour).Unix() }, "lifetime too long"},
		{"not yet valid", func(_, p map[string]any) { p["nbf"] = now.Add(time.Hour).Unix() }, "not yet valid"},
		{"missing jti", func(_, p map[string]any) { delete(p, "jti") }, "jti required"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var assertion string
			if tc.name == "malformed jwt" {
				assertion = "not-a-jwt"
			} else {
				h, p := base()
				tc.mutate(h, p)
				assertion = encodeClientAssertion(t, key, h, p)
			}
			_, err := VerifyClientAssertion(context.Background(), assertion, clientID, []string{audience}, keysFn, replay, now, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("error = %v, want containing %q", err, tc.wantError)
			}
		})
	}
}

func TestVerifyClientAssertionRejectsKeysFnError(t *testing.T) {
	const (
		clientID = "private-client"
		audience = "https://idp.example/token"
	)
	key, jwk := dpopTestKey(t)
	jwk["kid"] = "client-key"
	now := time.Now().UTC()
	assertion := encodeClientAssertion(t, key,
		map[string]any{"alg": "PS256", "kid": "client-key"},
		map[string]any{"iss": clientID, "sub": clientID, "aud": audience, "jti": "jti-y", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix()},
	)
	keysFn := func(_ context.Context, _ string) ([]map[string]any, error) {
		return nil, context.Canceled
	}
	_, err := VerifyClientAssertion(
		context.Background(), assertion, clientID, []string{audience}, keysFn,
		memory.NewClientAssertionReplayStore(), now, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "load jwks") {
		t.Fatalf("expected jwks-load error, got %v", err)
	}
}

func TestVerifyClientAssertionRequiresReplayStore(t *testing.T) {
	const (
		clientID = "private-client"
		audience = "https://idp.example/token"
	)
	key, jwk := dpopTestKey(t)
	jwk["kid"] = "client-key"
	now := time.Now().UTC()
	assertion := encodeClientAssertion(t, key,
		map[string]any{"alg": "PS256", "kid": "client-key"},
		map[string]any{"iss": clientID, "sub": clientID, "aud": audience, "jti": "jti-z", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix()},
	)
	keysFn := func(_ context.Context, _ string) ([]map[string]any, error) {
		return []map[string]any{jwk}, nil
	}
	_, err := VerifyClientAssertion(context.Background(), assertion, clientID, []string{audience}, keysFn, nil, now, nil)
	if err == nil || !strings.Contains(err.Error(), "replay store is not configured") {
		t.Fatalf("expected replay-store-required error, got %v", err)
	}
}
