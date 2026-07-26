package db_postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/saml/domain"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestSamlIdentityProviderProfileRepository(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	ctx := context.Background()
	profiles := &SamlIdentityProviderProfileRepository{Pool: db}
	serviceProviders := &SamlServiceProviderRepository{Pool: db}

	defaultProfile, err := profiles.EnsureDefaultIDPProfile(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !defaultProfile.IsDefault || defaultProfile.ProfileID != domain.DefaultIDPProfileID {
		t.Fatalf("default profile = %+v", defaultProfile)
	}

	dedicated := &domain.SamlIdentityProviderProfile{
		TenantID: tenant.ID, ProfileID: "dedicated-a", Name: "Dedicated A", Mode: domain.IDPProfileModeDedicated,
	}
	if err := profiles.SaveIDPProfile(ctx, dedicated); err != nil {
		t.Fatal(err)
	}
	if err := serviceProviders.Save(ctx, &domain.SamlServiceProvider{
		TenantID: tenant.ID, EntityID: "urn:sp:a", IDPProfileID: dedicated.ProfileID,
	}); err != nil {
		t.Fatal(err)
	}
	if err := serviceProviders.Save(ctx, &domain.SamlServiceProvider{
		TenantID: tenant.ID, EntityID: "urn:sp:b", IDPProfileID: dedicated.ProfileID,
	}); !errors.Is(err, domain.ErrDedicatedIDPProfileCardinality) {
		t.Fatalf("second dedicated binding error = %v", err)
	}
	if err := profiles.DeleteIDPProfile(ctx, tenant.ID, dedicated.ProfileID); !errors.Is(err, domain.ErrIDPProfileInUse) {
		t.Fatalf("delete in-use profile error = %v", err)
	}
	if err := profiles.DeleteIDPProfile(ctx, tenant.ID, domain.DefaultIDPProfileID); !errors.Is(err, domain.ErrDefaultIDPProfile) {
		t.Fatalf("delete default profile error = %v", err)
	}

	list, err := profiles.ListIDPProfilesByTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || !list[0].IsDefault {
		t.Fatalf("profiles = %+v", list)
	}
}
