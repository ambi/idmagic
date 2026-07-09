package postgres

import (
	"bytes"
	"context"
	"testing"

	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/tenancy"
)

func saltTenantCtx(id string) context.Context {
	return tenancy.WithTenant(context.Background(), &spec.Tenant{ID: id}, "https://issuer.example", "")
}

func TestTenantSaltStoreGeneratesAndIsStable(t *testing.T) {
	db := requireDB(t)
	store := NewTenantSaltStore(db)
	ctx := saltTenantCtx(newUUID(t))

	first, err := store.GetSalt(ctx)
	if err != nil {
		t.Fatalf("GetSalt: %v", err)
	}
	if len(first) == 0 {
		t.Fatal("generated salt is empty")
	}
	second, err := store.GetSalt(ctx)
	if err != nil {
		t.Fatalf("GetSalt (2nd): %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("salt changed on second read for same tenant (not idempotent)")
	}
}

func TestTenantSaltStoreSeparatesTenants(t *testing.T) {
	db := requireDB(t)
	store := NewTenantSaltStore(db)
	a, err := store.GetSalt(saltTenantCtx(newUUID(t)))
	if err != nil {
		t.Fatalf("GetSalt a: %v", err)
	}
	b, err := store.GetSalt(saltTenantCtx(newUUID(t)))
	if err != nil {
		t.Fatalf("GetSalt b: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("distinct tenants share the same salt")
	}
}
