package salts_postgres

import (
	"bytes"
	"context"
	"testing"

	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func saltTenantCtx(id string) context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: id}, "https://issuer.example", "")
}

func TestTenantSaltStoreGeneratesAndIsStable(t *testing.T) {
	db := pgtest.Require(t)
	store := NewTenantSaltStore(db)
	ctx := saltTenantCtx(pgfixtures.NewUUID(t))

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
	db := pgtest.Require(t)
	store := NewTenantSaltStore(db)
	a, err := store.GetSalt(saltTenantCtx(pgfixtures.NewUUID(t)))
	if err != nil {
		t.Fatalf("GetSalt a: %v", err)
	}
	b, err := store.GetSalt(saltTenantCtx(pgfixtures.NewUUID(t)))
	if err != nil {
		t.Fatalf("GetSalt b: %v", err)
	}
	if bytes.Equal(a, b) {
		t.Fatal("distinct tenants share the same salt")
	}
}
