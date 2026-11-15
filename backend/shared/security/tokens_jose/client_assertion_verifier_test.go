package tokens_jose

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

//spec:covers REQ-OAUTH2-028, RFC7518-SIGNATURE-ALGORITHMS: PS384 を独立に署名し、正当性と再送・誤署名の拒否を確認する。
func TestVerifyPS384ClientAssertion(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	const clientID, audience = "private-client", "https://idp.example/token"
	now := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
	header, err := json.Marshal(map[string]any{"alg": "PS384", "kid": "client-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": clientID, "sub": clientID, "aud": audience,
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "jti": "ps384-assertion",
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha512.Sum384([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA384, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	// 誤署名は同じ jti を消費しない。PS384 と名乗る SHA-256 の署名も拒否する。
	wrongDigest := sha256.Sum256([]byte(input))
	wrong, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, wrongDigest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	keys := func(_ context.Context, requestedClientID string) ([]map[string]any, error) {
		if requestedClientID != clientID {
			return nil, context.Canceled
		}
		jwk := rsaPublicJWK(&key.PublicKey)
		jwk["kid"] = "client-key"
		return []map[string]any{jwk}, nil
	}
	replay := newReplayStoreFake()
	verify := func(sig []byte) error {
		_, verifyErr := VerifyClientAssertion(t.Context(), input+"."+base64.RawURLEncoding.EncodeToString(sig),
			clientID, []string{audience}, keys, replay, now, nil)
		return verifyErr
	}
	if err := verify(wrong); err == nil {
		t.Fatal("PS384 と名乗る SHA-256 の署名が通った")
	}
	if err := verify(signature); err != nil {
		t.Fatalf("PS384 の assertion が拒否された: %v", err)
	}
	if err := verify(signature); err == nil {
		t.Fatal("同じ assertion が再利用できた")
	}
	proof := input + "." + base64.RawURLEncoding.EncodeToString(signature)
	if _, err := VerifyDPoPForToken(t.Context(), proof, "POST", audience, newReplayStoreFake(), now); err == nil {
		t.Fatal("PS384 が DPoP の受理方式へ混入した")
	}
}

//spec:covers REQ-OAUTH2-028: PS384 を追加しても ES256 のクライアントは同じ認証規則で扱う。
func TestVerifyClientAssertionPreservesES256(t *testing.T) {
	key, jwk := dpopTestECKey(t)
	jwk["kid"] = "ec-client-key"
	now := time.Now().UTC()
	claims := map[string]any{
		"iss": "private-client", "sub": "private-client", "aud": "https://idp.example/token",
		"iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "jti": "ec-assertion",
	}
	assertion := encodeECDPoPProof(t, key, map[string]any{"alg": "ES256", "kid": "ec-client-key"}, claims)
	keys := func(_ context.Context, clientID string) ([]map[string]any, error) {
		if clientID != "private-client" {
			return nil, context.Canceled
		}
		return []map[string]any{jwk}, nil
	}
	if _, err := VerifyClientAssertion(t.Context(), assertion, "private-client", []string{"https://idp.example/token"},
		keys, newReplayStoreFake(), now, nil); err != nil {
		t.Fatalf("ES256 の正当な assertion が拒否された: %v", err)
	}
}

func signClientAssertion(
	t *testing.T,
	key *rsa.PrivateKey,
	audience, jti string,
	now time.Time,
	ttl time.Duration,
) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": "client-key"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": "private-client", "sub": "private-client", "aud": audience, "jti": jti,
		"iat": now.Unix(), "exp": now.Add(ttl).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPSS(
		rand.Reader,
		key,
		crypto.SHA256,
		digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	)
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifyClientAssertion(t *testing.T) {
	const (
		clientID = "private-client"
		audience = "https://idp.example/token"
		kid      = "client-key"
	)
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaPublicJWK(&key.PublicKey)
	jwk["kid"] = kid
	keysFn := func(_ context.Context, requestedClientID string) ([]map[string]any, error) {
		if requestedClientID != clientID {
			return nil, context.Canceled
		}
		return []map[string]any{jwk}, nil
	}

	t.Run("accepts valid assertion and rejects replay", func(t *testing.T) {
		replay := newReplayStoreFake()
		assertion := signClientAssertion(t, key, audience, "jti-1", now, 2*time.Minute)
		if _, err := VerifyClientAssertion(
			context.Background(), assertion, clientID, []string{audience}, keysFn, replay, now, nil,
		); err != nil {
			t.Fatalf("valid assertion rejected: %v", err)
		}
		if _, err := VerifyClientAssertion(
			context.Background(), assertion, clientID, []string{audience}, keysFn, replay, now, nil,
		); err == nil {
			t.Fatal("expected replay rejection")
		}
	})

	t.Run("rejects wrong audience and excessive lifetime", func(t *testing.T) {
		replay := newReplayStoreFake()
		wrongAud := signClientAssertion(t, key, "https://other.example", "jti-2", now, time.Minute)
		if _, err := VerifyClientAssertion(
			context.Background(), wrongAud, clientID, []string{audience}, keysFn, replay, now, nil,
		); err == nil {
			t.Fatal("expected audience rejection")
		}
		longLived := signClientAssertion(t, key, audience, "jti-3", now, time.Hour)
		if _, err := VerifyClientAssertion(
			context.Background(), longLived, clientID, []string{audience}, keysFn, replay, now, nil,
		); err == nil {
			t.Fatal("expected lifetime rejection")
		}
	})

	t.Run("invalid signature does not reserve jti", func(t *testing.T) {
		replay := newReplayStoreFake()
		attacker, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		bad := signClientAssertion(t, attacker, audience, "shared-jti", now, time.Minute)
		if _, err := VerifyClientAssertion(
			context.Background(), bad, clientID, []string{audience}, keysFn, replay, now, nil,
		); err == nil {
			t.Fatal("expected signature rejection")
		}
		good := signClientAssertion(t, key, audience, "shared-jti", now, time.Minute)
		if _, err := VerifyClientAssertion(
			context.Background(), good, clientID, []string{audience}, keysFn, replay, now, nil,
		); err != nil {
			t.Fatalf("valid assertion rejected after bad signature: %v", err)
		}
	})
}
