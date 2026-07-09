package crypto

import (
	"bytes"
	"context"
	"testing"

	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"
)

func tenantCtx(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &spec.Tenant{ID: id}, "https://issuer.example", "")
}

func TestInMemoryTenantSaltStoreGeneratesOnFirstUse(t *testing.T) {
	store := NewInMemoryTenantSaltStore()
	salt, err := store.GetSalt(tenantCtx("tenant-a"))
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	if len(salt) != TenantSaltBytes {
		t.Fatalf("salt length = %d, want %d", len(salt), TenantSaltBytes)
	}
}

func TestInMemoryTenantSaltStoreIsStable(t *testing.T) {
	store := NewInMemoryTenantSaltStore()
	ctx := tenantCtx("tenant-a")
	first, _ := store.GetSalt(ctx)
	second, _ := store.GetSalt(ctx)
	if !bytes.Equal(first, second) {
		t.Fatal("salt changed between calls for the same tenant")
	}
}

func TestInMemoryTenantSaltStoreSeparatesTenants(t *testing.T) {
	store := NewInMemoryTenantSaltStore()
	a, _ := store.GetSalt(tenantCtx("tenant-a"))
	b, _ := store.GetSalt(tenantCtx("tenant-b"))
	if bytes.Equal(a, b) {
		t.Fatal("distinct tenants share the same salt")
	}
}

func TestInMemoryTenantSaltStoreEmptyTenantDefaults(t *testing.T) {
	// tenant 未設定 ctx は DefaultTenantID に解決され panic しない。
	store := NewInMemoryTenantSaltStore()
	salt, err := store.GetSalt(context.Background())
	if err != nil {
		t.Fatalf("GetSalt on empty ctx: %v", err)
	}
	if len(salt) != TenantSaltBytes {
		t.Fatalf("salt length = %d", len(salt))
	}
}
