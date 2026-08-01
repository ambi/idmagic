package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	oauthmemory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func secretClient(ctx context.Context, t *testing.T, repo *oauthmemory.OAuth2ClientRepository, now time.Time) *domain.OAuth2Client {
	t.Helper()
	hash := domain.HashClientSecret("legacy-secret")
	client := &domain.OAuth2Client{
		TenantID: "default", ClientID: "client", ClientSecretHash: &hash,
		ClientType: spec.ClientConfidential, GrantTypes: []spec.GrantType{spec.GrantClientCredentials},
		TokenEndpointAuthMethod:  domain.AuthMethodClientSecretBasic,
		IDTokenSignedResponseAlg: "PS256", FapiProfile: domain.FapiNone,
		CreatedAt: now.AddDate(0, -1, 0), UpdatedAt: now,
	}
	if err := repo.Save(ctx, client); err != nil {
		t.Fatal(err)
	}
	return client
}

func TestIssueClientSecretAddsExpiringCredentialWithoutChangingExisting(t *testing.T) {
	ctx := tenantContext("default")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	repo := oauthmemory.NewClientRepository()
	client := secretClient(ctx, t, repo, now)
	legacy := domain.ClientSecretCredential{
		CredentialID: "legacy", ClientID: client.ClientID,
		SecretHash: *client.ClientSecretHash, CreatedAt: client.CreatedAt,
	}
	if err := repo.SaveClientSecretCredential(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent

	result, err := IssueClientSecret(ctx, AdminOAuth2ClientDeps{
		ClientRepo: repo,
		Emit:       func(event spec.DomainEvent) { events = append(events, event) },
	}, IssueClientSecretInput{ActorUserID: "admin", ClientID: client.ClientID, ExpiresInDays: 90, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if result.ClientSecret == "" || result.Credential.ExpiresAt == nil ||
		!result.Credential.ExpiresAt.Equal(now.AddDate(0, 0, 90)) {
		t.Fatalf("unexpected issue result: %#v", result)
	}
	if len(result.Credentials) != 2 || result.Credentials[0].ExpiresAt != nil || result.Credentials[0].RevokedAt != nil {
		t.Fatalf("existing credential changed: %#v", result.Credentials)
	}
	if !domain.VerifyClientSecret(result.ClientSecret, result.Credential.SecretHash) {
		t.Fatal("issued secret was not stored as a hash")
	}
	if len(events) != 1 {
		t.Fatalf("events=%d, want 1", len(events))
	}
	event, ok := events[0].(*domain.ClientSecretIssued)
	if !ok || event.ClientID != client.ClientID || event.CredentialID != result.Credential.CredentialID ||
		!event.ExpiresAt.Equal(*result.Credential.ExpiresAt) {
		t.Fatalf("unexpected event: %#v", events[0])
	}
}

func TestIssueClientSecretValidatesExpiryAndActiveLimit(t *testing.T) {
	ctx := tenantContext("default")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

	for _, days := range []int{0, 731} {
		t.Run("invalid expiry", func(t *testing.T) {
			repo := oauthmemory.NewClientRepository()
			client := secretClient(ctx, t, repo, now)
			_, err := IssueClientSecret(ctx, AdminOAuth2ClientDeps{ClientRepo: repo}, IssueClientSecretInput{
				ClientID: client.ClientID, ExpiresInDays: days, Now: now,
			})
			if err == nil {
				t.Fatalf("expires_in_days=%d succeeded", days)
			}
		})
	}

	repo := oauthmemory.NewClientRepository()
	client := secretClient(ctx, t, repo, now)
	future := now.Add(time.Hour)
	for _, id := range []string{"one", "two"} {
		if err := repo.SaveClientSecretCredential(ctx, domain.ClientSecretCredential{
			CredentialID: id, ClientID: client.ClientID, SecretHash: domain.HashClientSecret(id),
			CreatedAt: now.Add(-time.Hour), ExpiresAt: &future,
		}); err != nil {
			t.Fatal(err)
		}
	}
	_, err := IssueClientSecret(ctx, AdminOAuth2ClientDeps{ClientRepo: repo}, IssueClientSecretInput{
		ClientID: client.ClientID, ExpiresInDays: 90, Now: now,
	})
	if !errors.Is(err, ErrClientSecretLimitExceeded) {
		t.Fatalf("error=%v, want ErrClientSecretLimitExceeded", err)
	}
}

func TestIssueClientSecretBackfillsLegacyCredential(t *testing.T) {
	ctx := tenantContext("default")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	repo := oauthmemory.NewClientRepository()
	client := secretClient(ctx, t, repo, now)

	result, err := IssueClientSecret(ctx, AdminOAuth2ClientDeps{ClientRepo: repo}, IssueClientSecretInput{
		ClientID: client.ClientID, ExpiresInDays: 30, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Credentials) != 2 || result.Credentials[0].ExpiresAt != nil ||
		result.Credentials[0].SecretHash != *client.ClientSecretHash {
		t.Fatalf("legacy credential was not backfilled: %#v", result.Credentials)
	}
}

func TestIssueClientSecretRejectsNonSecretClient(t *testing.T) {
	ctx := tenantContext("default")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	repo := oauthmemory.NewClientRepository()
	client := secretClient(ctx, t, repo, now)
	client.TokenEndpointAuthMethod = domain.AuthMethodPrivateKeyJwt

	_, err := IssueClientSecret(ctx, AdminOAuth2ClientDeps{ClientRepo: repo}, IssueClientSecretInput{
		ClientID: client.ClientID, ExpiresInDays: 90, Now: now,
	})
	if !errors.Is(err, ErrClientSecretNotManageable) {
		t.Fatalf("error=%v, want ErrClientSecretNotManageable", err)
	}
}

func TestRevokeClientSecretIsScopedAndIdempotent(t *testing.T) {
	ctx := tenantContext("default")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	repo := oauthmemory.NewClientRepository()
	client := secretClient(ctx, t, repo, now)
	if err := repo.SaveClientSecretCredential(ctx, domain.ClientSecretCredential{
		CredentialID: "old", ClientID: client.ClientID, SecretHash: domain.HashClientSecret("old"), CreatedAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	var events []spec.DomainEvent
	deps := AdminOAuth2ClientDeps{ClientRepo: repo, Emit: func(event spec.DomainEvent) { events = append(events, event) }}

	result, err := RevokeClientSecret(ctx, deps, RevokeClientSecretInput{
		ActorUserID: "admin", ClientID: client.ClientID, CredentialID: "old", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Credentials) != 1 || result.Credentials[0].RevokedAt == nil || !result.Credentials[0].RevokedAt.Equal(now) {
		t.Fatalf("unexpected credentials: %#v", result.Credentials)
	}
	if len(events) != 1 {
		t.Fatalf("events=%d, want 1", len(events))
	}
	event, ok := events[0].(*domain.ClientSecretRevoked)
	if !ok || event.CredentialID != "old" || event.ClientID != client.ClientID {
		t.Fatalf("unexpected event: %#v", events[0])
	}

	if _, err := RevokeClientSecret(ctx, deps, RevokeClientSecretInput{
		ActorUserID: "admin", ClientID: client.ClientID, CredentialID: "old", Now: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("idempotent revoke failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("idempotent revoke emitted duplicate event: %d", len(events))
	}

	_, err = RevokeClientSecret(ctx, deps, RevokeClientSecretInput{
		ActorUserID: "admin", ClientID: client.ClientID, CredentialID: "other", Now: now,
	})
	if !errors.Is(err, ErrClientSecretCredentialNotFound) {
		t.Fatalf("error=%v, want ErrClientSecretCredentialNotFound", err)
	}
}
