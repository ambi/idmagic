// Package testing_contract defines the shared WS-Federation persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/ambi/idmagic/backend/wsfederation/ports"
)

type Fixture struct {
	Repository ports.WsFedRelyingPartyRepository
	TenantA    string
	TenantB    string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, "missing"); err != nil || got != nil {
		t.Fatalf("missing FindByWtrealm = (%+v, %v)", got, err)
	}
	rp := &domain.WsFedRelyingParty{
		TenantID: f.TenantA, Wtrealm: "urn:example:rp", DisplayName: "Example RP",
		ReplyURLs: []string{"https://rp.example/callback"}, Audience: "urn:example:audience",
		TokenType: domain.TokenTypeSAML11, CreatedAt: f.Now, UpdatedAt: f.Now,
	}
	if err := f.Repository.Save(ctx, rp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, rp.Wtrealm)
	if err != nil || got == nil || got.DisplayName != rp.DisplayName || len(got.ReplyURLs) != 1 {
		t.Fatalf("FindByWtrealm = (%+v, %v), want round trip", got, err)
	}
	if leaked, err := f.Repository.FindByWtrealm(ctx, f.TenantB, rp.Wtrealm); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByWtrealm = (%+v, %v)", leaked, err)
	}
	second := *rp
	second.Wtrealm = "urn:example:second"
	if err := f.Repository.Save(ctx, &second); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	list, err := f.Repository.ListAll(ctx, f.TenantA)
	if err != nil || len(list) != 2 || list[0].Wtrealm != rp.Wtrealm || list[1].Wtrealm != second.Wtrealm {
		t.Fatalf("ListAll = (%+v, %v), want stable wtrealm order", list, err)
	}
	if err := f.Repository.Delete(ctx, f.TenantA, rp.Wtrealm); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, rp.Wtrealm); err != nil || got != nil {
		t.Fatalf("Find after Delete = (%+v, %v)", got, err)
	}
}
