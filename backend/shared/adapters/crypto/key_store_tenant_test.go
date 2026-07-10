package crypto_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/shared/adapters/crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

func tenantCtx(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &spec.Tenant{ID: id}, "", "")
}

// TenantJwksIsolation 不変条件: テナント指定 JWKS に載る鍵はすべて当該テナントに属する。
func TestInMemoryKeyStoreTenantIsolation(t *testing.T) {
	ks, err := crypto.NewInMemoryKeyStore()
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
	if keyA.Provider != spec.KeyProviderLocal {
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
	ks, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	ctxA := tenantCtx("tenant-a")
	first, err := ks.GetActiveKey(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ks.Rotate(ctxA)
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
