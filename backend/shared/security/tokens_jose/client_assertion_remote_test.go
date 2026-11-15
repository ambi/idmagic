package tokens_jose

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

//spec:covers REQ-OAUTH2-028: 外部 JWKS の取得・キャッシュ・鍵更新を実物の署名検証へ接続する。
func TestClientAssertionUsesRemoteJWKSAndExpiresCachedKeys(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var current atomic.Pointer[rsa.PrivateKey]
	current.Store(key)
	var hits atomic.Int64
	host := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Path != "/keys" {
			t.Errorf("鍵の取得先=%s", r.URL.Path)
		}
		jwk := rsaPublicJWK(&current.Load().PublicKey)
		jwk["kid"] = "client-key"
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{jwk}})
	}))
	t.Cleanup(host.Close)
	const uri = "https://192.0.2.1/keys"
	resolver := newStubResolver(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != uri {
			t.Errorf("JWKS の要求先=%s", req.URL)
		}
		routed := req.Clone(req.Context())
		var err error
		routed.URL, err = url.Parse(host.URL + req.URL.Path)
		if err != nil {
			return nil, err
		}
		return host.Client().Do(routed)
	})
	now := time.Now().UTC()
	resolver.now = func() time.Time { return now }
	keys := func(ctx context.Context, _ string) ([]map[string]any, error) {
		remote := uri
		return resolver.Resolve(ctx, JWKSource{JwksURI: &remote})
	}
	verify := func(key *rsa.PrivateKey, jti string) error {
		_, verifyErr := VerifyClientAssertion(t.Context(), signClientAssertion(t, key, "https://idp.example/token", jti, now, time.Minute),
			"private-client", []string{"https://idp.example/token"}, keys, newReplayStoreFake(), now, nil)
		return verifyErr
	}
	if err := verify(key, "remote-first"); err != nil {
		t.Fatal(err)
	}
	current.Store(rotated)
	if err := verify(rotated, "remote-cached"); err == nil {
		t.Fatal("キャッシュ内の別の鍵で署名を受理した")
	}
	if hits.Load() != 1 {
		t.Fatalf("キャッシュが再取得を防がない: hits=%d", hits.Load())
	}
	now = now.Add(6 * time.Minute)
	if err := verify(rotated, "remote-rotated"); err != nil {
		t.Fatalf("キャッシュ期限後の鍵を取得できない: %v", err)
	}
	if hits.Load() != 2 {
		t.Fatalf("期限切れ後の取得数=%d", hits.Load())
	}
}
