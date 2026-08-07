package verification_jose_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"testing"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/verification_jose"
)

func signSVID(t *testing.T, key *rsa.PrivateKey, kid, iss, sub, aud string, iat, exp time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": kid})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": iss, "sub": sub, "aud": aud, "iat": iat.Unix(), "exp": exp.Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestVerifier_Verify(t *testing.T) {
	const issuer = "https://issuer.example"
	const audience = "https://idmagic.example/token"
	const kid = "workload-key"
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := map[string]any{
		"kty": "RSA", "kid": kid,
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(key.E)).Bytes()),
	}
	fetch := func(context.Context) ([]map[string]any, error) { return []map[string]any{jwk}, nil } //nolint:unparam // error is always nil for this successful-fetch fixture; the failure path is exercised separately below

	v := verification_jose.NewVerifier()

	t.Run("accepts a valid SVID", func(t *testing.T) {
		token := signSVID(t, key, kid, issuer, "spiffe://example.org/ns/prod/sa/worker-1", audience, now, now.Add(10*time.Minute))
		claims, err := v.Verify(context.Background(), "bundle-1", token, issuer, []string{audience}, time.Hour, fetch, now)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if claims.Subject != "spiffe://example.org/ns/prod/sa/worker-1" {
			t.Fatalf("claims = %+v", claims)
		}
	})

	t.Run("translates spoofed signature to a domain sentinel error", func(t *testing.T) {
		attacker, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		token := signSVID(t, attacker, kid, issuer, "spiffe://example.org/ns/prod/sa/worker-1", audience, now, now.Add(10*time.Minute))
		_, err = v.Verify(context.Background(), "bundle-1", token, issuer, []string{audience}, time.Hour, fetch, now)
		if !errors.Is(err, workloaddomain.ErrSVIDInvalidSignature) {
			t.Fatalf("err = %v, want ErrSVIDInvalidSignature", err)
		}
	})

	t.Run("translates a fetch failure to ErrSVIDKeysUnavailable", func(t *testing.T) {
		failing := func(context.Context) ([]map[string]any, error) { return nil, errors.New("network down") }
		token := signSVID(t, key, kid, issuer, "spiffe://example.org/ns/prod/sa/worker-1", audience, now, now.Add(10*time.Minute))
		_, err := v.Verify(context.Background(), "bundle-2", token, issuer, []string{audience}, time.Hour, failing, now)
		if !errors.Is(err, workloaddomain.ErrSVIDKeysUnavailable) {
			t.Fatalf("err = %v, want ErrSVIDKeysUnavailable", err)
		}
	})
}
