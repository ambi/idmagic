package db_postgres

import (
	"bytes"
	"context"
	"testing"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func seedTestTenant(t *testing.T, db sharedpg.DB, id string) *domain.Tenant {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	tenant := &domain.Tenant{
		ID: id, Realm: "branding-" + id[len(id)-8:], DisplayName: "Branding Test Tenant",
		Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := (&TenantRepository{Pool: db}).Save(context.Background(), tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	return tenant
}

func TestTenantBrandingRepositorySaveAndFind(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTestTenant(t, db, "22222222-2222-2222-2222-222222222221")
	repo := &TenantBrandingRepository{Pool: db}
	ctx := context.Background()

	if existing, err := repo.FindByTenant(ctx, tenant.ID); err != nil || existing != nil {
		t.Fatalf("expected no branding row yet: %+v %v", existing, err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	branding := &domain.TenantBranding{
		TenantID:     tenant.ID,
		ProductName:  "Acme",
		PrimaryColor: "#0f172a",
		FooterLink1:  domain.TenantFooterLink{Label: "ヘルプ", URL: "https://support.example.com"},
		FooterText:   "(c) Acme",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repo.Save(ctx, branding); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByTenant(ctx, tenant.ID)
	if err != nil || got == nil {
		t.Fatalf("find: %v %+v", err, got)
	}
	if got.ProductName != "Acme" || got.PrimaryColor != "#0f172a" || got.FooterLink1.URL != "https://support.example.com" || got.FooterLink1.Label != "ヘルプ" {
		t.Fatalf("branding not round-tripped: %+v", got)
	}
	if got.LogoURL != "" || got.FooterLink2.IsSet() {
		t.Fatalf("expected unset fields to remain empty: %+v", got)
	}
}

func TestTenantBrandingAssetStoreRoundTrip(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTestTenant(t, db, "22222222-2222-2222-2222-222222222222")
	store := &TenantBrandingAssetStore{Pool: db}
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	asset := &domain.TenantBrandingAsset{
		TenantID:    tenant.ID,
		Kind:        domain.TenantBrandingAssetKindLogo,
		ObjectKey:   "logo-1",
		ContentType: "image/png",
		SizeBytes:   4,
		Data:        []byte{0x1, 0x2, 0x3, 0x4},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.Save(ctx, asset); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Find(ctx, tenant.ID, domain.TenantBrandingAssetKindLogo, "logo-1")
	if err != nil || got == nil {
		t.Fatalf("find: %v %+v", err, got)
	}
	if got.ContentType != "image/png" || !bytes.Equal(got.Data, asset.Data) {
		t.Fatalf("asset not round-tripped: %+v", got)
	}

	if err := store.DeleteByTenant(ctx, tenant.ID, domain.TenantBrandingAssetKindLogo); err != nil {
		t.Fatalf("delete by tenant: %v", err)
	}
	got, err = store.Find(ctx, tenant.ID, domain.TenantBrandingAssetKindLogo, "logo-1")
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}
