package db_memory

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/saml/domain"
)

func TestSamlIdentityProviderProfilesAndBindings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := NewSamlServiceProviderRepository()

	defaultProfile, err := repo.EnsureDefaultIDPProfile(ctx, "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if defaultProfile.ProfileID != domain.DefaultIDPProfileID || !defaultProfile.IsDefault {
		t.Fatalf("default profile = %+v", defaultProfile)
	}
	profiles, err := repo.ListIDPProfilesByTenant(ctx, "tenant-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("profiles = %d, want 1", len(profiles))
	}

	dedicated := &domain.SamlIdentityProviderProfile{
		TenantID: "tenant-a", ProfileID: "dedicated-a", Name: "Dedicated A", Mode: domain.IDPProfileModeDedicated,
	}
	if err := repo.SaveIDPProfile(ctx, dedicated); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, &domain.SamlServiceProvider{
		TenantID: "tenant-a", EntityID: "urn:sp:a", IDPProfileID: dedicated.ProfileID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, &domain.SamlServiceProvider{
		TenantID: "tenant-a", EntityID: "urn:sp:b", IDPProfileID: dedicated.ProfileID,
	}); !errors.Is(err, domain.ErrDedicatedIDPProfileCardinality) {
		t.Fatalf("second dedicated binding error = %v", err)
	}
	if err := repo.DeleteIDPProfile(ctx, "tenant-a", dedicated.ProfileID); !errors.Is(err, domain.ErrIDPProfileInUse) {
		t.Fatalf("delete in-use profile error = %v", err)
	}
	if err := repo.DeleteIDPProfile(ctx, "tenant-a", domain.DefaultIDPProfileID); !errors.Is(err, domain.ErrDefaultIDPProfile) {
		t.Fatalf("delete default profile error = %v", err)
	}

	shared := &domain.SamlIdentityProviderProfile{
		TenantID: "tenant-a", ProfileID: "shared", Name: "Shared", Mode: domain.IDPProfileModeShared,
	}
	if err := repo.SaveIDPProfile(ctx, shared); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteIDPProfile(ctx, "tenant-a", shared.ProfileID); err != nil {
		t.Fatal(err)
	}
	if found, err := repo.FindIDPProfileByID(ctx, "tenant-a", shared.ProfileID); err != nil || found != nil {
		t.Fatalf("deleted profile = %+v, err = %v", found, err)
	}
}
