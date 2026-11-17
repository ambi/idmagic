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
	"slices"
	"testing"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/verification_jose"
)

const (
	testIssuer   = "https://issuer.example"
	testAudience = "https://idmagic.example/token"
	testKID      = "workload-key"
	testSubject  = "spiffe://example.org/ns/prod/sa/worker-1"
)

// newSigningKey は署名鍵と、その公開鍵の JWK を返す。
func newSigningKey(t *testing.T) (*rsa.PrivateKey, map[string]any) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key, map[string]any{
		"kty": "RSA", "kid": testKID,
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(key.E)).Bytes()),
	}
}

func signSVID(t *testing.T, key *rsa.PrivateKey, iat, exp time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": testKID})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": testIssuer, "sub": testSubject, "aud": testAudience, "iat": iat.Unix(), "exp": exp.Unix(),
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

func fetchKeys(jwk map[string]any) func(context.Context) ([]map[string]any, error) {
	return func(context.Context) ([]map[string]any, error) { return []map[string]any{jwk}, nil }
}

func failingFetch(context.Context) ([]map[string]any, error) {
	return nil, errors.New("network down")
}

func TestVerifier_Verify(t *testing.T) {
	now := time.Now().UTC()
	key, jwk := newSigningKey(t)
	fetch := fetchKeys(jwk)

	v := verification_jose.NewVerifier(nil)

	t.Run("accepts a valid SVID", func(t *testing.T) {
		token := signSVID(t, key, now, now.Add(10*time.Minute))
		claims, err := v.Verify(context.Background(), oidcBundle("bundle-1"), token, fetch, now)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if claims.Subject != testSubject {
			t.Fatalf("claims = %+v", claims)
		}
	})

	t.Run("translates spoofed signature to a domain sentinel error", func(t *testing.T) {
		attacker, _ := newSigningKey(t)
		token := signSVID(t, attacker, now, now.Add(10*time.Minute))
		_, err := v.Verify(context.Background(), oidcBundle("bundle-1"), token, fetch, now)
		if !errors.Is(err, workloaddomain.ErrSVIDInvalidSignature) {
			t.Fatalf("err = %v, want ErrSVIDInvalidSignature", err)
		}
	})

	t.Run("translates a fetch failure to ErrSVIDKeysUnavailable", func(t *testing.T) {
		token := signSVID(t, key, now, now.Add(10*time.Minute))
		_, err := v.Verify(context.Background(), oidcBundle("bundle-2"), token, failingFetch, now)
		if !errors.Is(err, workloaddomain.ErrSVIDKeysUnavailable) {
			t.Fatalf("err = %v, want ErrSVIDKeysUnavailable", err)
		}
	})
}

// 取得に失敗して最後に取得できた鍵を使ったとき、出所を観測者へ渡す。メモリの鍵は
// jwks_uri ごとに持つので、jwks_uri を変えた信頼設定には古い取得元の鍵を使わない。
//
//spec:covers REQ-WORKLOADIDENTITY-018: 現在の jwks_uri から最後に取得できた鍵だけを使い、プロセスのメモリと保存した鍵のどちらを使ったかを区別する。
func TestVerifier_ReportsTheLastGoodKeySource(t *testing.T) {
	now := time.Now().UTC()
	key, jwk := newSigningKey(t)
	token := signSVID(t, key, now, now.Add(10*time.Minute))
	withURI := func(uri string) *workloaddomain.WorkloadTrustBundle {
		b := oidcBundle("bundle-1")
		b.JWKSURI = &uri
		return b
	}

	var sources []string
	v := verification_jose.NewVerifier(func(source string) { sources = append(sources, source) })
	if _, err := v.Verify(context.Background(), withURI("https://issuer.example/jwks"), token, fetchKeys(jwk), now); err != nil {
		t.Fatalf("live Verify: %v", err)
	}
	if _, err := v.Verify(context.Background(), withURI("https://issuer.example/jwks"), token, failingFetch, now.Add(time.Minute)); err != nil {
		t.Fatalf("memory Verify: %v", err)
	}
	if _, err := v.Verify(context.Background(), withURI("https://issuer.example/moved"), token, failingFetch, now.Add(time.Minute)); !errors.Is(err, workloaddomain.ErrSVIDKeysUnavailable) {
		t.Fatalf("moved source Verify err = %v, want ErrSVIDKeysUnavailable", err)
	}
	stored := withURI("https://issuer.example/moved")
	fetchedAt := now.Add(-time.Hour)
	stored.JWKSCached, stored.JWKSCachedAt = []map[string]any{jwk}, &fetchedAt
	if _, err := v.Verify(context.Background(), stored, token, failingFetch, now.Add(time.Minute)); err != nil {
		t.Fatalf("stored Verify: %v", err)
	}
	if want := []string{"memory", "stored"}; !slices.Equal(sources, want) {
		t.Fatalf("sources = %v, want %v", sources, want)
	}
}

func oidcBundle(id string) *workloaddomain.WorkloadTrustBundle {
	return &workloaddomain.WorkloadTrustBundle{
		ID: id, Profile: workloaddomain.WorkloadTrustBundleProfileOIDC, Issuer: testIssuer,
		AcceptedAudiences: []string{testAudience}, MaxSubjectTokenTTLSeconds: 3600,
	}
}
