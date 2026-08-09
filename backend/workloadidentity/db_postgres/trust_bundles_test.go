package db_postgres

import (
	"context"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestWorkloadTrustBundleRepository_SaveFindDelete(t *testing.T) {
	db := pgtest.Require(t)
	repo := &WorkloadTrustBundleRepository{Pool: db}
	tenant := seedTenant(t, db)

	b := newTrustBundle(t, tenant.ID)
	b.JWKS = map[string]any{"keys": []any{}}
	if err := repo.Save(context.Background(), b); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(context.Background(), tenant.ID, b.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil || got.Issuer != b.Issuer || got.Name != b.Name {
		t.Fatalf("FindByID = %+v, want match for %+v", got, b)
	}
	if got.JWKS == nil {
		t.Fatal("JWKS round-trip lost inline jwks")
	}

	byIssuer, err := repo.FindByIssuer(context.Background(), tenant.ID, b.Issuer)
	if err != nil {
		t.Fatalf("FindByIssuer: %v", err)
	}
	if byIssuer == nil || byIssuer.ID != b.ID {
		t.Fatalf("FindByIssuer = %+v, want %q", byIssuer, b.ID)
	}

	if err := repo.Delete(context.Background(), tenant.ID, b.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	gone, err := repo.FindByID(context.Background(), tenant.ID, b.ID)
	if err != nil {
		t.Fatalf("FindByID after delete: %v", err)
	}
	if gone != nil {
		t.Fatal("expected bundle to be gone after Delete")
	}
}

func TestWorkloadTrustBundleRepository_ListByTenantIsScoped(t *testing.T) {
	db := pgtest.Require(t)
	repo := &WorkloadTrustBundleRepository{Pool: db}
	tenantA := seedTenant(t, db)
	tenantB := seedTenant(t, db)

	a := newTrustBundle(t, tenantA.ID)
	if err := repo.Save(context.Background(), a); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	b := newTrustBundle(t, tenantB.ID)
	if err := repo.Save(context.Background(), b); err != nil {
		t.Fatalf("Save b: %v", err)
	}

	list, err := repo.ListAll(context.Background(), tenantA.ID)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(list) != 1 || list[0].ID != a.ID {
		t.Fatalf("ListAll(tenantA) = %+v, want only %q", list, a.ID)
	}
}

// TestWorkloadTrustBundleRepository_IssuerUniquePerTenant — DB constraint
// workload_trust_bundles_tenant_issuer_key が同一テナント内の issuer 重複登録を
// 拒否することを検証する (scenario `管理者がtrustbundleを登録・無効化・再有効化できる`
// の extension「同一テナント内に同じ issuer の WorkloadTrustBundle が既に存在する」)。
func TestWorkloadTrustBundleRepository_IssuerUniquePerTenant(t *testing.T) {
	db := pgtest.Require(t)
	repo := &WorkloadTrustBundleRepository{Pool: db}
	tenant := seedTenant(t, db)

	first := newTrustBundle(t, tenant.ID)
	if err := repo.Save(context.Background(), first); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	second := newTrustBundle(t, tenant.ID)
	second.Issuer = first.Issuer
	if err := repo.Save(context.Background(), second); err == nil {
		t.Fatal("expected duplicate issuer within the same tenant to be rejected")
	}
}
