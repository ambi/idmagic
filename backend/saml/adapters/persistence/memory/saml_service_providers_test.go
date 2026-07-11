package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/saml/domain"
)

func TestSamlServiceProviderRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewSamlServiceProviderRepository()

	t.Run("Save and FindByEntityID", func(t *testing.T) {
		sp := &domain.SamlServiceProvider{
			TenantID:      "tenant-1",
			EntityID:      "urn:sp-1",
			DisplayName:   "Service Provider 1",
			ACSURLs:       []string{"https://sp1.example.com/acs"},
			SignAssertion: true,
			CreatedAt:     time.Now(),
		}

		err := repo.Save(ctx, sp)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByEntityID(ctx, "tenant-1", "urn:sp-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected SP to be found")
		}
		if found.DisplayName != "Service Provider 1" {
			t.Errorf("expected DisplayName to be 'Service Provider 1', got %q", found.DisplayName)
		}
		if len(found.ACSURLs) != 1 || found.ACSURLs[0] != "https://sp1.example.com/acs" {
			t.Errorf("unexpected ACSURLs: %v", found.ACSURLs)
		}

		// 存在しない SP
		notfound, err := repo.FindByEntityID(ctx, "tenant-1", "urn:sp-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing SP")
		}
	})

	t.Run("Seed", func(t *testing.T) {
		sp := &domain.SamlServiceProvider{
			TenantID: "tenant-1",
			EntityID: "urn:sp-seeded",
		}
		//nolint:contextcheck // memory repo Seed doesn't take context
		repo.Seed(sp)

		found, err := repo.FindByEntityID(ctx, "tenant-1", "urn:sp-seeded")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected seeded SP to be found")
		}
	})

	t.Run("ListByTenant", func(t *testing.T) {
		// すでに urn:sp-1, urn:sp-seeded が tenant-1 に存在する
		spC := &domain.SamlServiceProvider{TenantID: "tenant-1", EntityID: "urn:sp-c"}
		spB := &domain.SamlServiceProvider{TenantID: "tenant-1", EntityID: "urn:sp-b"}
		spOther := &domain.SamlServiceProvider{TenantID: "tenant-other", EntityID: "urn:sp-other"}

		_ = repo.Save(ctx, spC)
		_ = repo.Save(ctx, spB)
		_ = repo.Save(ctx, spOther)

		list, err := repo.ListByTenant(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		// tenant-1 には 4 つあるはず
		if len(list) != 4 {
			t.Fatalf("expected 4 SPs, got %d", len(list))
		}
		// EntityID 順（urn:sp-1, urn:sp-b, urn:sp-c, urn:sp-seeded）でソートされていることを検証
		if list[0].EntityID != "urn:sp-1" || list[1].EntityID != "urn:sp-b" || list[2].EntityID != "urn:sp-c" || list[3].EntityID != "urn:sp-seeded" {
			t.Errorf("list is not sorted by EntityID: %s, %s, %s, %s", list[0].EntityID, list[1].EntityID, list[2].EntityID, list[3].EntityID)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "tenant-1", "urn:sp-1")
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByEntityID(ctx, "tenant-1", "urn:sp-1")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected SP to be deleted")
		}
	})
}
