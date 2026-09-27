// Package testing_contract defines the shared consent repository contract.
package testing_contract

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/consent/domain"
	"github.com/ambi/idmagic/backend/oauth2/consent/ports"
)

type Fixture struct {
	Repository ports.ConsentRepository
	TenantID   string
	Consent    *domain.Consent
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.Repository.Save(ctx, f.TenantID, f.Consent); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.Find(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID)
	if err != nil || got == nil || got.State != domain.ConsentGranted {
		t.Fatalf("Find = (%+v, %v)", got, err)
	}
	all, err := f.Repository.FindAll(ctx, f.TenantID)
	if err != nil || len(all) != 1 {
		t.Fatalf("FindAll = (%+v, %v)", all, err)
	}
	if err := f.Repository.Revoke(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	got, err = f.Repository.Find(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID)
	if err != nil || got == nil || got.State != domain.ConsentRevoked || got.RevokedAt == nil {
		t.Fatalf("Find after revoke = (%+v, %v)", got, err)
	}
	if err := f.Repository.DeleteAllForSub(ctx, f.Consent.UserID); err != nil {
		t.Fatalf("DeleteAllForSub: %v", err)
	}
}
