package db_postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/db_postgres"
	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestConnectionAndIdentityRepositoriesRoundTrip(t *testing.T) {
	pool := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), pgtest.Now()
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	identities := &db_postgres.IdentityRepository{Pool: pool}
	connection := testConnection(tenant.ID, now)
	if err := connections.Save(ctx, connection); err != nil {
		t.Fatalf("Save connection: %v", err)
	}
	if found, err := connections.Find(ctx, tenant.ID, connection.ID); err != nil || found == nil || found.SecretReference != connection.SecretReference {
		t.Fatalf("Find connection=(%+v,%v)", found, err)
	}

	userID := pgfixtures.NewUUID(t)
	user := &userdomain.User{
		ID: userID, TenantID: tenant.ID, PreferredUsername: "federated-user",
		PasswordHash: "!", Roles: []string{},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := (&userpg.UserRepository{Pool: pool}).Save(ctx, user); err != nil {
		t.Fatalf("Save user: %v", err)
	}
	link := &domain.FederatedIdentity{
		TenantID: tenant.ID, ProviderID: connection.ID, ExternalSubject: "external",
		LocalUserID: userID, LinkedAt: now,
	}
	if err := identities.Create(ctx, link); err != nil {
		t.Fatalf("Create link: %v", err)
	}
	conflict := *link
	conflict.ExternalSubject = "other"
	if err := identities.Create(ctx, &conflict); !errors.Is(err, federationports.ErrLinkConflict) {
		t.Fatalf("conflict err=%v", err)
	}
	if found, err := identities.FindBySubject(ctx, tenant.ID, connection.ID, "external"); err != nil || found == nil || found.LocalUserID != userID {
		t.Fatalf("FindBySubject=(%+v,%v)", found, err)
	}
}

func TestAttemptStoreConsumesOnceAndScopesTenant(t *testing.T) {
	pool := pgtest.Require(t)
	tenantA, tenantB := pgfixtures.SeedTenant(t, pool), pgfixtures.SeedTenant(t, pool)
	ctx, now := context.Background(), time.Now().UTC()
	connections := &db_postgres.ConnectionRepository{Pool: pool}
	if err := connections.Save(ctx, testConnection(tenantA.ID, now)); err != nil {
		t.Fatal(err)
	}
	attempts := &db_postgres.AttemptStore{Pool: pool}
	attempt := &domain.FederatedLoginAttempt{
		State: "state", TenantID: tenantA.ID, ProviderID: "oidc", Protocol: domain.ProtocolOIDC,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if err := attempts.Save(ctx, attempt); err != nil {
		t.Fatal(err)
	}
	if _, err := attempts.Consume(ctx, tenantB.ID, attempt.State, now); !errors.Is(err, federationports.ErrAttemptNotFound) {
		t.Fatalf("cross tenant err=%v", err)
	}
	if _, err := attempts.Consume(ctx, tenantA.ID, attempt.State, now); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	if _, err := attempts.Consume(ctx, tenantA.ID, attempt.State, now); !errors.Is(err, federationports.ErrAttemptConsumed) {
		t.Fatalf("second consume err=%v", err)
	}
}

func testConnection(tenantID string, now time.Time) *domain.IdentityProviderConnection {
	return &domain.IdentityProviderConnection{
		ID: "oidc", TenantID: tenantID, DisplayName: "OIDC",
		Protocol: domain.ProtocolOIDC, Status: domain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client", SecretReference: "env:OIDC_SECRET",
		AuthorizationEndpoint: "https://idp.example/auth", TokenEndpoint: "https://idp.example/token",
		JWKSURI:       "https://idp.example/jwks",
		ClaimMapping:  domain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: domain.LinkingNone, CreatedAt: now, UpdatedAt: now,
	}
}
