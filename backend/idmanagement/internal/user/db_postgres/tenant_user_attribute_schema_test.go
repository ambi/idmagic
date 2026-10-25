package db_postgres

import (
	"context"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/idmanagement/internal/user/testing_contract"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

//spec:covers REQ-IDMANAGEMENT-091: PostgreSQL に保存してもローカル専用と従来の省略値を区別できる。
func TestAttributeWriteAuthorityRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	testing_contract.AttributeWriteAuthorityRoundTrip(t, &TenantUserAttributeSchemaRepository{Pool: db}, tenant.ID)
}

func TestTenantUserAttributeSchemaRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &TenantUserAttributeSchemaRepository{Pool: db}
	ctx := context.Background()

	if got, err := repo.FindByTenant(ctx, tenant.ID); err != nil || got != nil {
		t.Fatalf("expected no schema initially: %v %+v", err, got)
	}

	now := testClock()
	schema := &userdomain.TenantUserAttributeSchema{
		TenantID: tenant.ID,
		Attributes: []userdomain.UserAttributeDef{
			{Key: "department", Label: "Department", Type: idmdomain.AttributeTypeString},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, schema); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByTenant(ctx, tenant.ID)
	if err != nil || got == nil || len(got.Attributes) != 1 {
		t.Fatalf("find by tenant: %v %+v", err, got)
	}
	if got.Attributes[0].Key != "department" {
		t.Fatalf("attributes not round-tripped: %+v", got.Attributes)
	}

	if err := repo.Delete(ctx, tenant.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByTenant(ctx, tenant.ID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}
