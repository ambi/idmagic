// Package testing_contract defines the shared workload-identity persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/ports"
)

type Fixture struct {
	TrustBundles ports.WorkloadTrustBundleRepository
	TenantA      string
	TenantB      string
	BundleID     string
	OtherID      string
	Now          time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	bundle := &domain.WorkloadTrustBundle{
		ID: f.BundleID, TenantID: f.TenantA, Name: "Production", TrustDomain: "example.test",
		Issuer: "https://issuer.example.test", JWKS: map[string]any{"keys": []any{}},
		AcceptedAudiences: []string{"idmagic"}, MaxSubjectTokenTTLSeconds: 300,
		Status: domain.WorkloadTrustBundleStatusEnabled, CreatedAt: f.Now,
	}
	if err := f.TrustBundles.Save(ctx, bundle); err != nil {
		t.Fatalf("Save: %v", err)
	}
	found, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID)
	if err != nil || found == nil || found.Issuer != bundle.Issuer || len(found.AcceptedAudiences) != 1 {
		t.Fatalf("FindByID = (%+v, %v)", found, err)
	}
	byIssuer, err := f.TrustBundles.FindByIssuer(ctx, f.TenantA, bundle.Issuer)
	if err != nil || byIssuer == nil || byIssuer.ID != f.BundleID {
		t.Fatalf("FindByIssuer = (%+v, %v)", byIssuer, err)
	}
	if leaked, err := f.TrustBundles.FindByID(ctx, f.TenantB, f.BundleID); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByID = (%+v, %v)", leaked, err)
	}
	listed, err := f.TrustBundles.ListAll(ctx, f.TenantA)
	if err != nil || len(listed) != 1 || listed[0].ID != f.BundleID {
		t.Fatalf("ListAll = (%+v, %v)", listed, err)
	}
	duplicate := *bundle
	duplicate.ID = f.OtherID
	if err := f.TrustBundles.Save(ctx, &duplicate); err == nil {
		t.Fatal("duplicate issuer within one tenant was accepted")
	}
	otherTenant := *bundle
	otherTenant.ID, otherTenant.TenantID = f.OtherID, f.TenantB
	if err := f.TrustBundles.Save(ctx, &otherTenant); err != nil {
		t.Fatalf("same issuer in another tenant: %v", err)
	}
	if err := f.TrustBundles.Delete(ctx, f.TenantA, f.BundleID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted, err := f.TrustBundles.FindByID(ctx, f.TenantA, f.BundleID); err != nil || deleted != nil {
		t.Fatalf("FindByID after Delete = (%+v, %v)", deleted, err)
	}
}
