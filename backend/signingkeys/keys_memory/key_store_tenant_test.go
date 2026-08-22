package keys_memory_test

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/ambi/idmagic/backend/tenancy"
)

func tenantCtx(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: id}, "", "")
}

// SCL scenario "XML federation署名資格情報はテナントと用途で分離される" の RED。
// REQ-SIGNINGKEYS-005: XML フェデレーション署名資格情報はテナントと用途で分離される。
func TestInMemoryKeyStoreSeparatesXMLFederationUsage(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")
	jwtKey, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	xmlCtx := signingports.WithKeyUsage(ctx, signingdomain.KeyUsageXMLFederationSigning)
	xmlKey, err := ks.GetActiveKey(xmlCtx)
	if err != nil {
		t.Fatal(err)
	}
	if jwtKey.Kid == xmlKey.Kid {
		t.Fatal("JWT and XML federation usages must have distinct active keys")
	}
	if len(jwtKey.CertificateDER) != 0 {
		t.Fatal("JWT key must not carry an X.509 certificate")
	}
	if _, err := x509.ParseCertificate(xmlKey.CertificateDER); err != nil {
		t.Fatalf("XML federation key certificate is invalid: %v", err)
	}
	xmlKeys, err := ks.ListPublicKeys(xmlCtx, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if len(xmlKeys) != 1 || xmlKeys[0].Usage != signingdomain.KeyUsageXMLFederationSigning {
		t.Fatalf("XML key list = %#v", xmlKeys)
	}
}

// SCL scenario "grace期間終了後の署名鍵はJWKSから除去されarchiveされる" の RED。
// REQ-SIGNINGKEYS-002: 猶予期間を過ぎた検証用鍵は JWKS から除かれる。
func TestInMemoryKeyStoreArchivesExpiredVerifyingKey(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")
	first, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ks.Rotate(ctx, time.Now().UTC(), 7*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	public, err := ks.ListPublicKeys(ctx, now)
	if err != nil || len(public) != 2 {
		t.Fatalf("grace JWKS: err=%v len=%d", err, len(public))
	}
	archived, err := ks.ArchiveExpired(ctx, now.Add(8*24*time.Hour))
	if err != nil || len(archived) != 1 || archived[0].Kid != first.Kid {
		t.Fatalf("archive expired: err=%v keys=%+v", err, archived)
	}
	public, err = ks.ListPublicKeys(ctx, now.Add(8*24*time.Hour))
	if err != nil || len(public) != 1 {
		t.Fatalf("expired key must be absent from JWKS: err=%v len=%d", err, len(public))
	}
}

// TenantJwksIsolation 不変条件: テナント指定 JWKS に載る鍵はすべて当該テナントに属する。
// REQ-SIGNINGKEYS-004: テナントごとの JWKS は互いに分離される。
func TestInMemoryKeyStoreTenantIsolation(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctxA, ctxB := tenantCtx("tenant-a"), tenantCtx("tenant-b")

	keyA, err := ks.GetActiveKey(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	keyB, err := ks.GetActiveKey(ctxB)
	if err != nil {
		t.Fatal(err)
	}
	if keyA.Kid == keyB.Kid {
		t.Fatalf("tenants must have distinct kids: %s", keyA.Kid)
	}
	if keyA.TenantID != "tenant-a" || keyB.TenantID != "tenant-b" {
		t.Fatalf("tenant ids mismatch: a=%s b=%s", keyA.TenantID, keyB.TenantID)
	}
	if keyA.Provider != signingdomain.KeyProviderLocal {
		t.Fatalf("provider=%s want Local", keyA.Provider)
	}

	jwksA, err := ks.GetAllKeys(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range jwksA {
		if k.Kid == keyB.Kid {
			t.Fatalf("tenant-a JWKS leaked tenant-b kid %s", keyB.Kid)
		}
		if k.TenantID != "tenant-a" {
			t.Fatalf("tenant-a JWKS contains foreign tenant key: %s", k.TenantID)
		}
	}
}

// Rotate は当該テナントの旧鍵を JWKS に残しつつ新鍵を active にする。
func TestInMemoryKeyStoreRotateKeepsTenantScope(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctxA := tenantCtx("tenant-a")
	first, err := ks.GetActiveKey(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ks.Rotate(ctxA, time.Now().UTC(), 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if first.Kid == second.Kid {
		t.Fatal("rotate must produce a new kid")
	}
	keys, err := ks.GetAllKeys(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("tenant-a should have 2 keys after rotate, got %d", len(keys))
	}
	// Disable で漏洩疑いの鍵を JWKS から即時に外す。
	if _, err := ks.Disable(ctxA, first.Kid); err != nil {
		t.Fatal(err)
	}
	keys, err = ks.GetAllKeys(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0].Kid != second.Kid {
		t.Fatalf("disable should leave only the active key, got %d", len(keys))
	}
}

// REQ-SIGNINGKEYS-010: 現在の署名鍵は即時無効化できない。
func TestInMemoryKeyStoreRejectsDisablingActiveKey(t *testing.T) {
	ks, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenantCtx("tenant-a")
	active, err := ks.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ks.Disable(ctx, active.Kid); !errors.Is(err, signingdomain.ErrActiveSigningKeyCannotBeDisabled) {
		t.Fatalf("Disable(active) error = %v, want ErrActiveSigningKeyCannotBeDisabled", err)
	}
}
