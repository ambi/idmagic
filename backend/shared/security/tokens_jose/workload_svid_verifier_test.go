package tokens_jose

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

const testWorkloadSubject = "spiffe://example.org/ns/prod/sa/worker-1"

func signWorkloadSVID(
	t *testing.T,
	key *rsa.PrivateKey,
	alg, iss, aud string,
	iat, exp time.Time,
) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": alg, "kid": "workload-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": iss, "sub": testWorkloadSubject, "aud": aud,
		"iat": iat.Unix(), "exp": exp.Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	var sig []byte
	switch alg {
	case "RS256":
		sig, err = rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	case "PS256":
		sig, err = rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	default:
		t.Fatalf("unsupported test alg %q", alg)
	}
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyWorkloadSVID(t *testing.T) {
	const (
		issuer   = "https://issuer.example"
		audience = "https://idmagic.example/token"
		kid      = "workload-key"
		subject  = "spiffe://example.org/ns/prod/sa/worker-1"
	)
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaPublicJWK(&key.PublicKey)
	jwk["kid"] = kid
	jwks := []map[string]any{jwk}

	t.Run("accepts a valid RS256 workload SVID", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, audience, now, now.Add(10*time.Minute))
		claims, err := VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now)
		if err != nil {
			t.Fatalf("valid SVID rejected: %v", err)
		}
		if claims.Subject != subject {
			t.Fatalf("Subject = %q, want %q", claims.Subject, subject)
		}
		if claims.Issuer != issuer {
			t.Fatalf("Issuer = %q, want %q", claims.Issuer, issuer)
		}
	})

	t.Run("rejects spoofed signature", func(t *testing.T) {
		attacker, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		token := signWorkloadSVID(t, attacker, "RS256", issuer, audience, now, now.Add(10*time.Minute))
		_, err = VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now)
		if !errors.Is(err, ErrWorkloadSVIDInvalidSignature) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDInvalidSignature", err)
		}
	})

	t.Run("rejects unregistered issuer", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", "https://unknown-issuer.example", audience, now, now.Add(10*time.Minute))
		_, err := VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now)
		if !errors.Is(err, ErrWorkloadSVIDIssuerMismatch) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDIssuerMismatch", err)
		}
	})

	t.Run("rejects audience mismatch", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, "https://other.example", now, now.Add(10*time.Minute))
		_, err := VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now)
		if !errors.Is(err, ErrWorkloadSVIDAudienceMismatch) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDAudienceMismatch", err)
		}
	})

	t.Run("rejects expired token", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, audience, now.Add(-2*time.Hour), now.Add(-time.Hour))
		_, err := VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now)
		if !errors.Is(err, ErrWorkloadSVIDExpired) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDExpired", err)
		}
	})

	t.Run("rejects a token whose lifetime exceeds the trust bundle max TTL", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "RS256", issuer, audience, now, now.Add(2*time.Hour))
		_, err := VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now)
		if !errors.Is(err, ErrWorkloadSVIDTTLExceeded) {
			t.Fatalf("err = %v, want ErrWorkloadSVIDTTLExceeded", err)
		}
	})

	t.Run("accepts PS256 as well as RS256", func(t *testing.T) {
		token := signWorkloadSVID(t, key, "PS256", issuer, audience, now, now.Add(10*time.Minute))
		if _, err := VerifyWorkloadSVID(token, jwks, issuer, []string{audience}, time.Hour, now); err != nil {
			t.Fatalf("PS256 SVID rejected: %v", err)
		}
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		_, err := VerifyWorkloadSVID("not-a-jwt", jwks, issuer, []string{audience}, time.Hour, now)
		if err == nil {
			t.Fatal("expected error for malformed token")
		}
	})
}

func TestWorkloadJWKSCache(t *testing.T) {
	now := time.Now().UTC()
	goodKeys := []map[string]any{{"kid": "k1"}}

	t.Run("returns freshly fetched keys and remembers them", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		calls := 0
		fetch := func(context.Context) ([]map[string]any, error) {
			calls++
			return goodKeys, nil
		}
		keys, stale, err := cache.Get(context.Background(), "bundle-1", fetch, now)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if stale {
			t.Fatal("freshly fetched keys must not be reported stale")
		}
		if len(keys) != 1 || calls != 1 {
			t.Fatalf("keys=%v calls=%d", keys, calls)
		}
	})

	t.Run("falls back to last-known-good within the staleness window on fetch failure", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		fetch := func(context.Context) ([]map[string]any, error) { return goodKeys, nil }
		if _, _, err := cache.Get(context.Background(), "bundle-1", fetch, now); err != nil {
			t.Fatalf("seed fetch: %v", err)
		}
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		keys, stale, err := cache.Get(context.Background(), "bundle-1", failing, now.Add(time.Hour))
		if err != nil {
			t.Fatalf("expected fallback to last-known-good, got error: %v", err)
		}
		if !stale {
			t.Fatal("fallback keys must be reported stale")
		}
		if len(keys) != 1 {
			t.Fatalf("keys=%v", keys)
		}
	})

	t.Run("fails closed once the last-known-good entry exceeds the staleness window", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		fetch := func(context.Context) ([]map[string]any, error) { return goodKeys, nil }
		if _, _, err := cache.Get(context.Background(), "bundle-1", fetch, now); err != nil {
			t.Fatalf("seed fetch: %v", err)
		}
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		_, _, err := cache.Get(context.Background(), "bundle-1", failing, now.Add(25*time.Hour))
		if err == nil {
			t.Fatal("expected fail-closed once last-known-good is stale beyond the window")
		}
	})

	t.Run("fails closed with no prior successful fetch", func(t *testing.T) {
		cache := NewWorkloadJWKSCache()
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		_, _, err := cache.Get(context.Background(), "bundle-1", failing, now)
		if err == nil {
			t.Fatal("expected fail-closed with no cache entry")
		}
	})
}

func TestAudienceStrings(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []string
	}{
		{"single string", "aud-1", []string{"aud-1"}},
		{"string array", []any{"a", "b"}, []string{"a", "b"}},
		{"array with non-string entries", []any{"a", 1, false}, []string{"a"}},
		{"empty array", []any{}, []string{}},
		{"unsupported type", 42, nil},
		{"nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := audienceStrings(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("audienceStrings(%#v) = %#v, want %#v", tc.in, got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("audienceStrings(%#v) = %#v, want %#v", tc.in, got, tc.want)
				}
			}
		})
	}
}
