package tokens_jose

import (
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

func signSecurityEventToken(t *testing.T, key *rsa.PrivateKey, alg, iss, aud, jti string, iat time.Time, events map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": alg, "kid": "set-key", "typ": "secevent+jwt"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": iss, "aud": aud, "jti": jti, "iat": iat.Unix(), "events": events,
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
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

func TestVerifySecurityEventToken(t *testing.T) {
	const (
		issuer   = "https://transmitter.example"
		audience = "https://idmagic.example/ssf/streams/stream-1/events"
		kid      = "set-key"
	)
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaPublicJWK(&key.PublicKey)
	jwk["kid"] = kid
	jwks := []map[string]any{jwk}
	events := map[string]any{"https://schemas.openid.net/secevent/caep/event-type/session-revoked": map[string]any{"subject": map[string]any{"subject_type": "Agent"}}}

	t.Run("accepts a validly-signed RS256 SET", func(t *testing.T) {
		token := signSecurityEventToken(t, key, "RS256", issuer, audience, "jti-1", now, events)
		claims, err := VerifySecurityEventToken(token, jwks, issuer, []string{audience})
		if err != nil {
			t.Fatalf("valid SET rejected: %v", err)
		}
		if claims.JTI != "jti-1" || claims.Issuer != issuer {
			t.Fatalf("unexpected claims: %+v", claims)
		}
		if _, ok := claims.Payload["events"]; !ok {
			t.Fatalf("expected events claim in payload: %+v", claims.Payload)
		}
	})

	t.Run("accepts PS256", func(t *testing.T) {
		token := signSecurityEventToken(t, key, "PS256", issuer, audience, "jti-2", now, events)
		if _, err := VerifySecurityEventToken(token, jwks, issuer, []string{audience}); err != nil {
			t.Fatalf("PS256 SET rejected: %v", err)
		}
	})

	t.Run("rejects spoofed signature", func(t *testing.T) {
		attacker, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		token := signSecurityEventToken(t, attacker, "RS256", issuer, audience, "jti-3", now, events)
		if _, err := VerifySecurityEventToken(token, jwks, issuer, []string{audience}); !errors.Is(err, ErrSecurityEventTokenInvalidSignature) {
			t.Fatalf("err = %v, want ErrSecurityEventTokenInvalidSignature", err)
		}
	})

	t.Run("rejects unregistered issuer", func(t *testing.T) {
		token := signSecurityEventToken(t, key, "RS256", "https://unknown.example", audience, "jti-4", now, events)
		if _, err := VerifySecurityEventToken(token, jwks, issuer, []string{audience}); !errors.Is(err, ErrSecurityEventTokenIssuerMismatch) {
			t.Fatalf("err = %v, want ErrSecurityEventTokenIssuerMismatch", err)
		}
	})

	t.Run("rejects audience mismatch", func(t *testing.T) {
		token := signSecurityEventToken(t, key, "RS256", issuer, "https://other.example", "jti-5", now, events)
		if _, err := VerifySecurityEventToken(token, jwks, issuer, []string{audience}); !errors.Is(err, ErrSecurityEventTokenAudienceMismatch) {
			t.Fatalf("err = %v, want ErrSecurityEventTokenAudienceMismatch", err)
		}
	})

	t.Run("rejects a missing jti", func(t *testing.T) {
		token := signSecurityEventToken(t, key, "RS256", issuer, audience, "", now, events)
		if _, err := VerifySecurityEventToken(token, jwks, issuer, []string{audience}); !errors.Is(err, ErrSecurityEventTokenMissingClaim) {
			t.Fatalf("err = %v, want ErrSecurityEventTokenMissingClaim", err)
		}
	})

	t.Run("rejects malformed token", func(t *testing.T) {
		if _, err := VerifySecurityEventToken("not-a-jwt", jwks, issuer, []string{audience}); err == nil {
			t.Fatal("expected error for malformed token")
		}
	})
}
