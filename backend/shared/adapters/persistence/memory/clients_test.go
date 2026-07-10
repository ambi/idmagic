package memory

import (
	"context"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestOAuth2ClientRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewClientRepository()

	t.Run("Save and FindByID", func(t *testing.T) {
		name := "My Application"
		secret := "secret-hash"
		client := &oauthdomain.OAuth2Client{
			TenantID:                "tenant-1",
			ClientID:                "client-1",
			ClientSecretHash:        &secret,
			ClientName:              &name,
			ClientType:              spec.ClientConfidential,
			RedirectURIs:            []string{"https://example.com/callback"},
			GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
			TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
			CreatedAt:               time.Now(),
		}

		err := repo.Save(ctx, client)
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected client to be found")
		}
		if found.ClientID != "client-1" {
			t.Errorf("expected ClientID to be 'client-1', got %q", found.ClientID)
		}
		if found.ClientName == nil || *found.ClientName != "My Application" {
			t.Errorf("expected ClientName to be 'My Application', got %v", found.ClientName)
		}

		// 存在しないクライアント
		notfound, err := repo.FindByID(ctx, "tenant-1", "client-none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil for non-existing client")
		}
	})

	t.Run("Seed", func(t *testing.T) {
		client := &oauthdomain.OAuth2Client{
			TenantID: "tenant-1",
			ClientID: "client-seeded",
		}
		//nolint:contextcheck // memory repo Seed doesn't take context
		repo.Seed(client)

		found, err := repo.FindByID(ctx, "tenant-1", "client-seeded")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected seeded client to be found")
		}
	})

	t.Run("FindAll", func(t *testing.T) {
		// すでに client-1, client-seeded が tenant-1 に存在する
		clientOther := &oauthdomain.OAuth2Client{
			TenantID: "tenant-other",
			ClientID: "client-other",
		}
		_ = repo.Save(ctx, clientOther)

		list, err := repo.FindAll(ctx, "tenant-1")
		if err != nil {
			t.Fatal(err)
		}
		// tenant-1 には client-1, client-seeded の 2 つがあるはず
		if len(list) != 2 {
			t.Fatalf("expected 2 clients, got %d", len(list))
		}

		// tenant-other には client-other の 1 つがあるはず
		listOther, err := repo.FindAll(ctx, "tenant-other")
		if err != nil {
			t.Fatal(err)
		}
		if len(listOther) != 1 || listOther[0].ClientID != "client-other" {
			t.Errorf("expected client-other, got %v", listOther)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "tenant-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}

		found, err := repo.FindByID(ctx, "tenant-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if found != nil {
			t.Error("expected client-1 to be deleted")
		}
	})
}
