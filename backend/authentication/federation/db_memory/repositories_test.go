package db_memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
)

func TestRepositoriesEnforceTenantAndLinkUniqueness(t *testing.T) {
	ctx := context.Background()
	repos := NewRepositories()
	now := time.Now().UTC()
	connection := &domain.IdentityProviderConnection{
		ID: "provider", TenantID: "tenant-a", DisplayName: "Provider",
		Protocol: domain.ProtocolOIDC, Status: domain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  domain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: domain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Connections.Save(ctx, connection); err != nil {
		t.Fatalf("Save connection: %v", err)
	}
	if got, _ := repos.Connections.Find(ctx, "tenant-b", "provider"); got != nil {
		t.Fatal("connection must not cross tenant boundary")
	}

	first := &domain.FederatedIdentity{
		TenantID: "tenant-a", ProviderID: "provider", ExternalSubject: "external",
		LocalUserID: "user-1", LinkedAt: now,
	}
	if err := repos.Identities.Create(ctx, first); err != nil {
		t.Fatalf("Create first link: %v", err)
	}
	for _, conflict := range []*domain.FederatedIdentity{
		{TenantID: "tenant-a", ProviderID: "provider", ExternalSubject: "external", LocalUserID: "user-2", LinkedAt: now},
		{TenantID: "tenant-a", ProviderID: "provider", ExternalSubject: "other", LocalUserID: "user-1", LinkedAt: now},
	} {
		if err := repos.Identities.Create(ctx, conflict); !errors.Is(err, federationports.ErrLinkConflict) {
			t.Fatalf("conflict err=%v, want ErrLinkConflict", err)
		}
	}
}

func TestAttemptStoreConsumesAtomically(t *testing.T) {
	ctx := context.Background()
	repos := NewRepositories()
	now := time.Now().UTC()
	attempt := &domain.FederatedLoginAttempt{
		State: "state", TenantID: "tenant-a", ProviderID: "provider", Protocol: domain.ProtocolOIDC,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := repos.Attempts.Save(ctx, attempt); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := repos.Attempts.Consume(ctx, "tenant-b", "state", now); !errors.Is(err, federationports.ErrAttemptNotFound) {
		t.Fatalf("cross tenant err=%v", err)
	}
	if _, err := repos.Attempts.Consume(ctx, "tenant-a", "state", now); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if _, err := repos.Attempts.Consume(ctx, "tenant-a", "state", now); !errors.Is(err, federationports.ErrAttemptConsumed) {
		t.Fatalf("second consume err=%v", err)
	}
}
