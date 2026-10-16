package db_memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestOAuth2ClientRepository(t *testing.T) {
	ctx := context.Background()
	repo := NewClientRepository()

	t.Run("Save and FindByID", func(t *testing.T) {
		name := "My Application"
		secret := "secret-hash"
		client := &domain.OAuth2Client{
			TenantID:                "tenant-1",
			ClientID:                "client-1",
			ClientSecretHash:        &secret,
			ClientName:              &name,
			ClientType:              spec.ClientConfidential,
			RedirectURIs:            []string{"https://example.com/callback"},
			GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
			TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
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
		client := &domain.OAuth2Client{
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
		clientOther := &domain.OAuth2Client{
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

func TestIssueClientSecretCredentialEnforcesActiveLimitAtomically(t *testing.T) {
	ctx := context.Background()
	repo := NewClientRepository()
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	legacy := domain.ClientSecretCredential{
		ID: "legacy", ClientID: "client", SecretHash: "legacy-hash", CreatedAt: now.Add(-time.Hour),
	}
	if err := repo.SaveClientSecretCredential(ctx, legacy); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, id := range []string{"issued-1", "issued-2"} {
		go func(credentialID string) {
			ready.Done()
			<-start
			errs <- repo.IssueClientSecretCredential(ctx, nil, domain.ClientSecretCredential{
				ID: credentialID, ClientID: "client", SecretHash: credentialID, CreatedAt: now,
			}, 2, now)
		}(id)
	}
	ready.Wait()
	close(start)

	succeeded := 0
	limited := 0
	for range 2 {
		err := <-errs
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ports.ErrClientSecretCredentialLimitExceeded):
			limited++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if succeeded != 1 || limited != 1 {
		t.Fatalf("succeeded=%d limited=%d", succeeded, limited)
	}
	credentials, err := repo.ListClientSecretCredentials(ctx, "client")
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials) != 2 {
		t.Fatalf("credentials=%d, want 2", len(credentials))
	}
}

func TestOAuth2ClientRepositoryListPage(t *testing.T) {
	ctx := context.Background()
	repo := NewClientRepository()
	for _, id := range []string{"c3", "c1", "c2", "c4", "c5"} {
		if err := repo.Save(ctx, &domain.OAuth2Client{TenantID: "tenant-1", ClientID: id, ClientType: spec.ClientConfidential}); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.Save(ctx, &domain.OAuth2Client{TenantID: "tenant-2", ClientID: "other", ClientType: spec.ClientConfidential}); err != nil {
		t.Fatal(err)
	}

	first, err := repo.ListPage(ctx, "tenant-1", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || first[0].ClientID != "c1" || first[1].ClientID != "c2" {
		t.Fatalf("unexpected first page: %+v", first)
	}
	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, "tenant-1", last.ClientID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(next) != 2 || next[0].ClientID != "c3" || next[1].ClientID != "c4" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}
	all, err := repo.ListPage(ctx, "tenant-1", "", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
