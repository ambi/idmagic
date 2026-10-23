package bootstrap

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func vaultRotationServer(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	keys := make([]*rsa.PrivateKey, 2)
	for i := range keys {
		var err error
		keys[i], err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
	}
	version := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var data any
		switch {
		case r.URL.Path == "/v1/sys/health":
			w.WriteHeader(200)
			return
		case strings.HasSuffix(r.URL.Path, "/rotate"):
			version++
			if version > len(keys) {
				w.WriteHeader(500)
				return
			}
			w.WriteHeader(204)
			return
		case strings.Contains(r.URL.Path, "/keys/"):
			der, err := x509.MarshalPKIXPublicKey(&keys[version-1].PublicKey)
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			data = map[string]any{"latest_version": version, "keys": map[string]any{fmt.Sprint(version): map[string]string{"public_key": string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))}}}
		case strings.Contains(r.URL.Path, "/sign/"):
			var body struct {
				Input     string `json:"input"`
				Version   int    `json:"key_version"`
				Algorithm string `json:"signature_algorithm"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			digest, err := base64.StdEncoding.DecodeString(body.Input)
			if err != nil {
				t.Error(err)
				w.WriteHeader(400)
				return
			}
			if body.Version < 1 || body.Version > len(keys) {
				w.WriteHeader(400)
				return
			}
			var signature []byte
			if body.Algorithm == "pss" {
				signature, err = rsa.SignPSS(rand.Reader, keys[body.Version-1], crypto.SHA256, digest, nil)
			} else {
				signature, err = rsa.SignPKCS1v15(rand.Reader, keys[body.Version-1], crypto.SHA256, digest)
			}
			if err != nil {
				t.Error(err)
				w.WriteHeader(500)
				return
			}
			data = map[string]string{"signature": fmt.Sprintf("vault:v%d:%s", body.Version, base64.StdEncoding.EncodeToString(signature))}
		default:
			w.WriteHeader(404)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

//spec:covers REQ-SIGNINGKEYS-007: 本番と共通の Assemble が PostgreSQL と Vault の構成を選び、再構築後も準備したバージョンを有効にしないこと。
func TestAssembleKeepsTheVaultXMLSelectionInPostgres(t *testing.T) {
	db := pgtest.Require(t)
	server := vaultRotationServer(t)
	loader := NewConfigLoader(stubEnv(map[string]string{"PERSISTENCE": "postgres", "DATABASE_URL": db.Config().ConnString(), "KEY_PROVIDER": "vault", "VAULT_ADDR": server.URL, "VAULT_TOKEN": "test-token"}))
	cfg := LoadSharedConfig(loader)
	if err := loader.Err(); err != nil {
		t.Fatal(err)
	}
	first, err := Assemble(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	ctx := signingports.WithKeyUsage(tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", ""), signingdomain.KeyUsageXMLFederationSigning)
	old, err := first.SigningKeys.KeyStore.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged := first.SigningKeys.KeyStore.(signingports.FederationKeyStore)
	prepared, err := staged.Prepare(ctx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	second, err := Assemble(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	current, err := second.SigningKeys.KeyStore.GetActiveKey(ctx)
	if err != nil || current.Kid != old.Kid {
		t.Fatalf("Assemble lost the active selection: %v %+v", err, current)
	}
	if current.Provider != signingdomain.KeyProviderVaultTransit {
		t.Fatalf("KEY_PROVIDER=vault selected %s", current.Provider)
	}
	digest := sha256.Sum256([]byte("the active key still signs through Vault after restart"))
	signer := current.PrivateKey.(crypto.Signer)
	signature, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	if err := rsa.VerifyPKCS1v15(old.PublicKey.(*rsa.PublicKey), crypto.SHA256, digest[:], signature); err != nil {
		t.Fatal(err)
	}
	cert, err := second.SigningKeys.FederationSigner().PreparedCertificate(ctx)
	if err != nil || cert == nil || base64.StdEncoding.EncodeToString(cert.Raw) != base64.StdEncoding.EncodeToString(prepared.Next.CertificateDER) {
		t.Fatal("Assemble lost the published candidate")
	}
	server.Close()
	if _, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256); err == nil {
		t.Fatal("signing succeeded with Vault unavailable")
	}
}
