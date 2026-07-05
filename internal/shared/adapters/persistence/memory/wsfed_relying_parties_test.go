package memory

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/internal/shared/spec"
)

func TestWsFedRelyingPartyRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewWsFedRelyingPartyRepository()

	rp := &spec.WsFedRelyingParty{
		Wtrealm:   "urn:federation:MicrosoftOnline",
		ReplyURLs: []string{"https://login.microsoftonline.com/login.srf"},
	}
	repo.Seed(rp) // tenant_id 未設定 → default に正規化される。

	got, err := repo.FindByWtrealm(ctx, spec.DefaultTenantID, "urn:federation:MicrosoftOnline")
	if err != nil {
		t.Fatalf("FindByWtrealm: %v", err)
	}
	if got == nil || got.TenantID != spec.DefaultTenantID {
		t.Fatalf("expected RP under default tenant, got %+v", got)
	}

	// クローンが返り、内部状態が共有されないこと。
	got.ReplyURLs[0] = "https://evil.example"
	again, _ := repo.FindByWtrealm(ctx, spec.DefaultTenantID, "urn:federation:MicrosoftOnline")
	if again.ReplyURLs[0] != "https://login.microsoftonline.com/login.srf" {
		t.Fatal("repository returned shared slice; mutation leaked")
	}

	missing, err := repo.FindByWtrealm(ctx, spec.DefaultTenantID, "urn:unknown")
	if err != nil {
		t.Fatalf("FindByWtrealm(unknown): %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for unknown wtrealm, got %+v", missing)
	}

	list, err := repo.ListByTenant(ctx, spec.DefaultTenantID)
	if err != nil {
		t.Fatalf("ListByTenant: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByTenant = %d entries, want 1", len(list))
	}

	// EntraProfile ありのクローン確認
	profile := spec.EntraFederationProfile{
		Domain: "example.com",
	}
	rpWithProfile := &spec.WsFedRelyingParty{
		Wtrealm:      "urn:federation:Entra",
		EntraProfile: &profile,
	}
	_ = repo.Save(ctx, rpWithProfile)

	gotProfile, _ := repo.FindByWtrealm(ctx, spec.DefaultTenantID, "urn:federation:Entra")
	if gotProfile.EntraProfile == nil || gotProfile.EntraProfile.Domain != "example.com" {
		t.Errorf("expected EntraProfile to be cloned correctly, got %v", gotProfile.EntraProfile)
	}

	// ソート順確認のためさらに追加
	rpAnother := &spec.WsFedRelyingParty{
		Wtrealm: "urn:federation:Another",
	}
	_ = repo.Save(ctx, rpAnother)

	listSorted, _ := repo.ListByTenant(ctx, spec.DefaultTenantID)
	// DefaultTenantID には urn:federation:MicrosoftOnline, urn:federation:Entra, urn:federation:Another の 3 件がある
	if len(listSorted) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(listSorted))
	}
	// ソート順（Another, Entra, MicrosoftOnline）の確認
	if listSorted[0].Wtrealm != "urn:federation:Another" || listSorted[1].Wtrealm != "urn:federation:Entra" || listSorted[2].Wtrealm != "urn:federation:MicrosoftOnline" {
		t.Errorf("expected list sorted by wtrealm: %s, %s, %s", listSorted[0].Wtrealm, listSorted[1].Wtrealm, listSorted[2].Wtrealm)
	}

	// Delete のテスト
	err = repo.Delete(ctx, spec.DefaultTenantID, "urn:federation:Another")
	if err != nil {
		t.Fatal(err)
	}
	deleted, _ := repo.FindByWtrealm(ctx, spec.DefaultTenantID, "urn:federation:Another")
	if deleted != nil {
		t.Error("expected RP to be deleted")
	}
}
