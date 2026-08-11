package db_postgres

import (
	"context"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestTenantGroupAttributeSchemaRepositoryRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &TenantGroupAttributeSchemaRepository{Pool: db}
	ctx := context.Background()

	if got, err := repo.FindByTenant(ctx, tenant.ID); err != nil || got != nil {
		t.Fatalf("expected no schema initially: %v %+v", err, got)
	}

	now := testClock()
	schema := &groupdomain.TenantGroupAttributeSchema{
		TenantID: tenant.ID,
		Attributes: []groupdomain.GroupAttributeDef{
			{Key: "cost_center", Label: "Cost Center", Type: idmdomain.AttributeTypeString},
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
	if got.Attributes[0].Key != "cost_center" {
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
