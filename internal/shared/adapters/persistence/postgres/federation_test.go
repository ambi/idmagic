package postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestSamlServiceProviderRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	repo := &SamlServiceProviderRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	sp := &spec.SamlServiceProvider{
		TenantID:      tenant.ID,
		EntityID:      "urn:sp:example",
		DisplayName:   "Example SP",
		ACSURLs:       []string{"https://sp.example/acs"},
		Audience:      "urn:sp:example",
		ClaimPolicy:   spec.ClaimMappingPolicy{NameID: spec.NameIdConfiguration{}},
		SignAssertion: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := repo.Save(ctx, sp); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByEntityID(ctx, tenant.ID, sp.EntityID)
	if err != nil || got == nil {
		t.Fatalf("find by entity id: %v %+v", err, got)
	}
	if got.DisplayName != "Example SP" || !got.SignAssertion {
		t.Fatalf("unexpected sp: %+v", got)
	}
	if len(got.ACSURLs) != 1 || got.ACSURLs[0] != "https://sp.example/acs" {
		t.Fatalf("acs urls not round-tripped: %+v", got.ACSURLs)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, tenant.ID, sp.EntityID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByEntityID(ctx, tenant.ID, sp.EntityID)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}

func TestWsFedRelyingPartyRepositoryRoundTrip(t *testing.T) {
	db := requireDB(t)
	tenant := seedTenant(t, db)
	repo := &WsFedRelyingPartyRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	rp := &spec.WsFedRelyingParty{
		TenantID:    tenant.ID,
		Wtrealm:     "urn:rp:example",
		DisplayName: "Example RP",
		ReplyURLs:   []string{"https://rp.example/wsfed"},
		Audience:    "urn:rp:example",
		TokenType:   spec.TokenTypeSAML11,
		ClaimPolicy: spec.ClaimMappingPolicy{NameID: spec.NameIdConfiguration{}},
		EntraProfile: &spec.EntraFederationProfile{
			Domain:                "example.com",
			IssuerURI:             "https://sts.example/adfs/services/trust",
			SourceAnchorAttribute: "objectGUID",
			ImmutableIDAttribute:  "objectGUID",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Save(ctx, rp); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.FindByWtrealm(ctx, tenant.ID, rp.Wtrealm)
	if err != nil || got == nil {
		t.Fatalf("find by wtrealm: %v %+v", err, got)
	}
	if got.DisplayName != "Example RP" || got.TokenType != spec.TokenTypeSAML11 {
		t.Fatalf("unexpected rp: %+v", got)
	}
	if len(got.ReplyURLs) != 1 || got.ReplyURLs[0] != "https://rp.example/wsfed" {
		t.Fatalf("reply urls not round-tripped: %+v", got.ReplyURLs)
	}
	if got.EntraProfile == nil || got.EntraProfile.Domain != "example.com" {
		t.Fatalf("entra profile not round-tripped: %+v", got.EntraProfile)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list by tenant: %v len=%d", err, len(list))
	}

	if err := repo.Delete(ctx, tenant.ID, rp.Wtrealm); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, err = repo.FindByWtrealm(ctx, tenant.ID, rp.Wtrealm)
	if err != nil || got != nil {
		t.Fatalf("expected deleted: %v %+v", err, got)
	}
}
