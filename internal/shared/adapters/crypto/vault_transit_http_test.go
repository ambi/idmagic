package crypto_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	adaptercrypto "github.com/ambi/idmagic/internal/shared/adapters/crypto"
)

func TestHTTPTransitEngineLatestPublicKeyAndSign(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))

	var gotToken string
	var signBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Vault-Token")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/transit/keys/idmagic-signing-default":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"latest_version": 1,
					"keys":           map[string]any{"1": map[string]any{"public_key": pubPEM}},
				},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/transit/sign/idmagic-signing-default":
			_ = json.NewDecoder(r.Body).Decode(&signBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"signature": "vault:v1:" + base64.StdEncoding.EncodeToString([]byte("sig"))},
			})
		case r.URL.Path == "/v1/sys/health":
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	engine := adaptercrypto.NewHTTPTransitEngine(srv.URL, "test-token", "transit")
	ctx := context.Background()

	pemStr, version, err := engine.LatestPublicKey(ctx, "idmagic-signing-default")
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 || pemStr != pubPEM {
		t.Fatalf("version=%d pem match=%v", version, pemStr == pubPEM)
	}

	sig, err := engine.Sign(ctx, "idmagic-signing-default", 1, []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if string(sig) != "sig" {
		t.Fatalf("signature decode mismatch: %q", sig)
	}
	if gotToken != "test-token" {
		t.Fatalf("X-Vault-Token not sent: %q", gotToken)
	}
	if signBody["prehashed"] != true || signBody["signature_algorithm"] != "pss" {
		t.Fatalf("sign request must be prehashed pss: %+v", signBody)
	}
	if !engine.Healthy(ctx) {
		t.Fatal("engine must report healthy on 200 sys/health")
	}
}
