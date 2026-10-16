package db_memory

import (
	"context"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// 実装がポートを満たすことをコンパイル時に保証する。
var _ tenantports.TenantUserAttributeSchemaRepository = (*TenantUserAttributeSchemaRepository)(nil)

func TestTenantUserAttributeSchemaRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewTenantUserAttributeSchemaRepository()

	if got, err := repo.FindByTenant(ctx, "acme"); err != nil || got != nil {
		t.Fatalf("expected nil schema for unknown tenant, got %v, %v", got, err)
	}

	claim := "region"
	schema := &userdomain.TenantUserAttributeSchema{
		TenantID: "acme",
		Attributes: []userdomain.UserAttributeDef{
			{Key: "region", Type: idmdomain.AttributeTypeString, Required: true, Visibility: idmdomain.AttrVisibilityClaimExposed, ClaimName: &claim},
		},
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.Save(ctx, schema); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := repo.FindByTenant(ctx, "acme")
	if err != nil || got == nil {
		t.Fatalf("expected stored schema, got %v, %v", got, err)
	}
	if len(got.Attributes) != 1 || got.Attributes[0].Key != "region" {
		t.Fatalf("unexpected schema: %+v", got)
	}

	// 返却値の変更が保管値に波及しないこと (深いコピー)。
	got.Attributes[0].Key = "mutated"
	again, _ := repo.FindByTenant(ctx, "acme")
	if again.Attributes[0].Key != "region" {
		t.Fatal("stored schema must not alias returned slice")
	}

	// 既存スキーマがある状態で再 Save 時に CreatedAt が保持されることの検証
	createdTime := time.Now().Add(-1 * time.Hour).UTC()
	schemaWithCreated := &userdomain.TenantUserAttributeSchema{
		TenantID:  "acme",
		CreatedAt: createdTime,
	}
	_ = repo.Save(ctx, schemaWithCreated)

	schemaUpdate := &userdomain.TenantUserAttributeSchema{
		TenantID:  "acme",
		CreatedAt: time.Now().UTC(), // 新しい時間
	}
	_ = repo.Save(ctx, schemaUpdate)

	gotUpdated, _ := repo.FindByTenant(ctx, "acme")
	if !gotUpdated.CreatedAt.Equal(createdTime) {
		t.Errorf("expected CreatedAt to be preserved as %v, got %v", createdTime, gotUpdated.CreatedAt)
	}

	if err := repo.Delete(ctx, "acme"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if got, _ := repo.FindByTenant(ctx, "acme"); got != nil {
		t.Fatal("expected schema removed after delete")
	}
}

func TestTenantUserAttributeSchemaRepositoryDefaultsTenant(t *testing.T) {
	ctx := context.Background()
	repo := NewTenantUserAttributeSchemaRepository()
	if err := repo.Save(ctx, &userdomain.TenantUserAttributeSchema{UpdatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	got, err := repo.FindByTenant(ctx, "")
	if err != nil || got == nil {
		t.Fatalf("expected default-tenant schema, got %v, %v", got, err)
	}
	if got.TenantID != tenancydomain.DefaultTenantID {
		t.Fatalf("expected tenant_id %q, got %q", tenancydomain.DefaultTenantID, got.TenantID)
	}
}
