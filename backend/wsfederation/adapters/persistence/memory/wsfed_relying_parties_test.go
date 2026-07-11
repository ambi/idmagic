package memory

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/wsfederation/domain"
)

func TestWsFedRelyingPartyRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewWsFedRelyingPartyRepository()
	rp := &domain.WsFedRelyingParty{Wtrealm: "urn:federation:MicrosoftOnline", ReplyURLs: []string{"https://login.microsoftonline.com/login.srf"}}
	repo.Seed(rp)

	got, err := repo.FindByWtrealm(ctx, spec.DefaultTenantID, rp.Wtrealm)
	if err != nil || got == nil || got.TenantID != spec.DefaultTenantID {
		t.Fatalf("FindByWtrealm = %+v, %v", got, err)
	}
	got.ReplyURLs[0] = "https://evil.example"
	again, _ := repo.FindByWtrealm(ctx, spec.DefaultTenantID, rp.Wtrealm)
	if again.ReplyURLs[0] != "https://login.microsoftonline.com/login.srf" {
		t.Fatal("mutation leaked from returned clone")
	}

	if err := repo.Save(ctx, &domain.WsFedRelyingParty{Wtrealm: "urn:federation:Another"}); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListByTenant(ctx, spec.DefaultTenantID)
	if err != nil || len(list) != 2 || list[0].Wtrealm != "urn:federation:Another" {
		t.Fatalf("ListByTenant = %+v, %v", list, err)
	}
	if err := repo.Delete(ctx, spec.DefaultTenantID, "urn:federation:Another"); err != nil {
		t.Fatal(err)
	}
}
