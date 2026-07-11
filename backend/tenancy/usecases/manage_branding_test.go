package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestGetBrandingReturnsEmptyForUnconfiguredTenant(t *testing.T) {
	repo := tenancymemory.NewTenantBrandingRepository()
	branding, err := GetBranding(context.Background(), repo, domain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if branding == nil || branding.TenantID != domain.DefaultTenantID || branding.IsConfigured() {
		t.Fatalf("expected empty unconfigured branding, got %#v", branding)
	}
}

func TestUpdateBrandingPersistsAndClearsFields(t *testing.T) {
	repo := tenancymemory.NewTenantBrandingRepository()
	ctx := context.Background()
	now := time.Now().UTC()

	saved, err := UpdateBranding(ctx, repo, domain.DefaultTenantID, BrandingUpdateInput{
		ProductName:  new("Acme"),
		PrimaryColor: new("#0f172a"),
		SupportURL:   new("https://support.example.com"),
		FooterText:   new("(c) Acme"),
	}, now)
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if saved.ProductName != "Acme" || saved.PrimaryColor != "#0f172a" || saved.SupportURL != "https://support.example.com" {
		t.Fatalf("unexpected saved branding: %#v", saved)
	}
	if !saved.IsConfigured() {
		t.Fatal("expected configured branding")
	}

	cleared, err := UpdateBranding(ctx, repo, domain.DefaultTenantID, BrandingUpdateInput{
		ProductName: new(""),
	}, now)
	if err != nil {
		t.Fatalf("clear failed: %v", err)
	}
	if cleared.ProductName != "" {
		t.Fatalf("expected product_name cleared, got %q", cleared.ProductName)
	}
	if cleared.PrimaryColor != "#0f172a" {
		t.Fatalf("expected untouched fields to survive partial update, got %#v", cleared)
	}
}

func TestUpdateBrandingRejectsNonHTTPSLinks(t *testing.T) {
	repo := tenancymemory.NewTenantBrandingRepository()
	cases := []string{"javascript:alert(1)", "http://example.com", "data:text/html,<script>1</script>"}
	for _, link := range cases {
		if _, err := UpdateBranding(context.Background(), repo, domain.DefaultTenantID, BrandingUpdateInput{
			SupportURL: new(link),
		}, time.Now().UTC()); !errors.Is(err, ErrInvalidBranding) {
			t.Fatalf("link %q: expected ErrInvalidBranding, got %v", link, err)
		}
	}
}

func TestUpdateBrandingRejectsLowContrastColor(t *testing.T) {
	repo := tenancymemory.NewTenantBrandingRepository()
	if _, err := UpdateBranding(context.Background(), repo, domain.DefaultTenantID, BrandingUpdateInput{
		PrimaryColor: new("#eeeeee"),
	}, time.Now().UTC()); !errors.Is(err, ErrInvalidBranding) {
		t.Fatalf("expected ErrInvalidBranding for low-contrast color, got %v", err)
	}
}

func TestUploadAndDeleteBrandingAssetRoundTrip(t *testing.T) {
	brandingRepo := tenancymemory.NewTenantBrandingRepository()
	assetStore := tenancymemory.NewTenantBrandingAssetStore()
	ctx := context.Background()
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}

	branding, err := UploadBrandingAsset(ctx, brandingRepo, assetStore, domain.DefaultTenantID, UploadBrandingAssetInput{
		Kind: domain.TenantBrandingAssetKindLogo, Data: png, URL: "/tenant-branding-assets/logo/abc", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if branding.LogoObjectKey == "" || branding.LogoURL != "/tenant-branding-assets/logo/abc" {
		t.Fatalf("unexpected branding after upload: %#v", branding)
	}

	stored, err := assetStore.Find(ctx, domain.DefaultTenantID, domain.TenantBrandingAssetKindLogo, branding.LogoObjectKey)
	if err != nil || stored == nil || stored.ContentType != "image/png" {
		t.Fatalf("expected stored png asset, got %#v err=%v", stored, err)
	}

	deleted, err := DeleteBrandingAsset(ctx, brandingRepo, assetStore, domain.DefaultTenantID, domain.TenantBrandingAssetKindLogo, time.Now().UTC())
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if deleted.LogoObjectKey != "" || deleted.LogoURL != "" {
		t.Fatalf("expected logo reference cleared, got %#v", deleted)
	}
	if stillStored, _ := assetStore.Find(ctx, domain.DefaultTenantID, domain.TenantBrandingAssetKindLogo, branding.LogoObjectKey); stillStored != nil {
		t.Fatal("expected asset blob deleted")
	}
}

func TestUploadBrandingAssetRejectsSVG(t *testing.T) {
	brandingRepo := tenancymemory.NewTenantBrandingRepository()
	assetStore := tenancymemory.NewTenantBrandingAssetStore()
	_, err := UploadBrandingAsset(context.Background(), brandingRepo, assetStore, domain.DefaultTenantID, UploadBrandingAssetInput{
		Kind: domain.TenantBrandingAssetKindLogo, Data: []byte("<svg onload=alert(1)></svg>"), Now: time.Now().UTC(),
	})
	if !errors.Is(err, ErrBrandingAssetFormat) {
		t.Fatalf("expected ErrBrandingAssetFormat, got %v", err)
	}
}

func TestUploadBrandingAssetRejectsInvalidKind(t *testing.T) {
	brandingRepo := tenancymemory.NewTenantBrandingRepository()
	assetStore := tenancymemory.NewTenantBrandingAssetStore()
	_, err := UploadBrandingAsset(context.Background(), brandingRepo, assetStore, domain.DefaultTenantID, UploadBrandingAssetInput{
		Kind: "logo-mark", Data: []byte{0x89, 'P', 'N', 'G'}, Now: time.Now().UTC(),
	})
	if !errors.Is(err, ErrInvalidBrandingAssetKind) {
		t.Fatalf("expected ErrInvalidBrandingAssetKind, got %v", err)
	}
}

func TestDeleteBrandingAssetIsIdempotentForUnconfiguredTenant(t *testing.T) {
	brandingRepo := tenancymemory.NewTenantBrandingRepository()
	assetStore := tenancymemory.NewTenantBrandingAssetStore()
	branding, err := DeleteBrandingAsset(context.Background(), brandingRepo, assetStore, domain.DefaultTenantID, domain.TenantBrandingAssetKindFavicon, time.Now().UTC())
	if err != nil {
		t.Fatalf("expected no-op delete to succeed, got %v", err)
	}
	if branding.IsConfigured() {
		t.Fatalf("expected unconfigured branding, got %#v", branding)
	}
}
