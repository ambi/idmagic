// Package testing_contract defines the shared OAuth2 client repository contract.
package testing_contract

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
)

type Fixture struct {
	Repository ports.OAuth2ClientRepository
	Client     *domain.OAuth2Client
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.Repository.Save(ctx, f.Client); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.FindByID(ctx, f.Client.TenantID, f.Client.ClientID)
	if err != nil || got == nil || got.ClientID != f.Client.ClientID {
		t.Fatalf("FindByID = (%+v, %v)", got, err)
	}
	all, err := f.Repository.FindAll(ctx, f.Client.TenantID)
	if err != nil || len(all) != 1 || all[0].ClientID != f.Client.ClientID {
		t.Fatalf("FindAll = (%+v, %v)", all, err)
	}
	if got, err := f.Repository.FindByID(ctx, "00000000-0000-0000-0000-0000000000ff", f.Client.ClientID); err != nil || got != nil {
		t.Fatalf("cross-tenant FindByID = (%+v, %v)", got, err)
	}
	if err := f.Repository.Delete(ctx, f.Client.TenantID, f.Client.ClientID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.FindByID(ctx, f.Client.TenantID, f.Client.ClientID); err != nil || got != nil {
		t.Fatalf("FindByID after delete = (%+v, %v)", got, err)
	}
}
