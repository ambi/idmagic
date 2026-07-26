package keys_vault_test

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"sync"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"

	adaptercrypto "github.com/ambi/idmagic/backend/signingkeys/keys_vault"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func tenantCtx() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "tenant-a"}, "", "")
}

// fakeTransit は Vault Transit を in-memory の RSA 鍵で模擬する。
// 秘密鍵は fake の内側 (= Vault 相当) に留まり、Sign 経由でのみ使う。
type fakeTransit struct {
	mu       sync.Mutex
	keys     map[string][]*rsa.PrivateKey // name -> versions (1-indexed)
	healthy  bool
	signCall int
}

func newFakeTransit() *fakeTransit {
	return &fakeTransit{keys: map[string][]*rsa.PrivateKey{}, healthy: true}
}

func (f *fakeTransit) EnsureKey(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.keys[name]) == 0 {
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		f.keys[name] = []*rsa.PrivateKey{priv}
	}
	return nil
}

func (f *fakeTransit) RotateKey(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	f.keys[name] = append(f.keys[name], priv)
	return nil
}

func (f *fakeTransit) LatestPublicKey(_ context.Context, name string) (string, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	versions := f.keys[name]
	if len(versions) == 0 {
		return "", 0, errors.New("no key")
	}
	version := len(versions)
	der, err := x509.MarshalPKIXPublicKey(&versions[version-1].PublicKey)
	if err != nil {
		return "", 0, err
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return pemStr, version, nil
}

func (f *fakeTransit) Sign(_ context.Context, name string, version int, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.signCall++
	versions := f.keys[name]
	if version < 1 || version > len(versions) {
		return nil, errors.New("bad version")
	}
	if pss, ok := opts.(*rsa.PSSOptions); ok {
		return rsa.SignPSS(rand.Reader, versions[version-1], crypto.SHA256, digest, pss)
	}
	return rsa.SignPKCS1v15(rand.Reader, versions[version-1], crypto.SHA256, digest)
}

func TestVaultKeyStoreSeparatesXMLFederationUsage(t *testing.T) {
	engine := newFakeTransit()
	ks := adaptercrypto.NewVaultKeyStore(engine, "idmagic-signing-")
	ctx := tenantCtx()
	jwtKey, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	xmlKey, err := ks.GetActiveKey(signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning))
	if err != nil {
		t.Fatal(err)
	}
	if jwtKey.Kid == xmlKey.Kid || len(xmlKey.CertificateDER) == 0 {
		t.Fatalf("JWT/XML separation failed: jwt=%s xml=%s cert=%d", jwtKey.Kid, xmlKey.Kid, len(xmlKey.CertificateDER))
	}
}

// SCL scenario "XML federation署名資格情報は再起動後も同一である"。
func TestVaultXMLFederationCertificateIsStableAcrossStoreRestart(t *testing.T) {
	engine := newFakeTransit()
	ctx := signingports.WithKeyUsage(tenantCtx(), signingdomain.KeyUsageXMLFederationSigning)
	firstStore := adaptercrypto.NewVaultKeyStore(engine, "idmagic-signing-")
	first, err := firstStore.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secondStore := adaptercrypto.NewVaultKeyStore(engine, "idmagic-signing-")
	second, err := secondStore.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.CertificateDER, second.CertificateDER) {
		t.Fatal("certificate must be reproducible from the same durable Vault key")
	}
}

func (f *fakeTransit) Healthy(context.Context) bool { return f.healthy }

func TestVaultKeyStoreSignsViaEngineAndVerifies(t *testing.T) {
	engine := newFakeTransit()
	ks := adaptercrypto.NewVaultKeyStore(engine, "idmagic-signing-")
	ctx := tenantCtx()

	key, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if key.Provider != signingdomain.KeyProviderVaultTransit {
		t.Fatalf("provider=%s want VaultTransit", key.Provider)
	}
	// 秘密鍵はアプリ側に無い (Vault 内)。crypto.Signer 経由で署名を委譲する。
	signer, ok := key.PrivateKey.(crypto.Signer)
	if !ok {
		t.Fatal("vault key must expose a crypto.Signer")
	}
	digest := make([]byte, 32)
	sig, err := signer.Sign(rand.Reader, digest, &rsa.PSSOptions{Hash: crypto.SHA256, SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	pub, ok := key.PublicKey.(*rsa.PublicKey)
	if !ok {
		t.Fatal("public key must be RSA")
	}
	if err := rsa.VerifyPSS(pub, crypto.SHA256, digest, sig, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}); err != nil {
		t.Fatalf("signature from vault engine must verify: %v", err)
	}
	if engine.signCall == 0 {
		t.Fatal("signing must be delegated to the engine")
	}
}

func TestVaultKeyStoreRotateAdvancesVersion(t *testing.T) {
	engine := newFakeTransit()
	ks := adaptercrypto.NewVaultKeyStore(engine, "idmagic-signing-")
	ctx := tenantCtx()

	first, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ks.Rotate(ctx, time.Now().UTC(), 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if first.Kid == second.Kid {
		t.Fatal("rotate must produce a new kid backed by a new vault version")
	}
	keys, err := ks.GetAllKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("both versions must remain in JWKS, got %d", len(keys))
	}
}

func TestVaultKeyStoreFailClosedWhenEngineDown(t *testing.T) {
	engine := newFakeTransit()
	ks := adaptercrypto.NewVaultKeyStore(engine, "idmagic-signing-")
	ctx := tenantCtx()
	engine.healthy = false
	if ks.Healthy(ctx) {
		t.Fatal("key store must report unhealthy when the engine is down")
	}
}
