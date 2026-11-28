package db_postgres_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/internal/federation/db_postgres"
	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	"github.com/ambi/idmagic/backend/authentication/internal/federation/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		now := pgfixtures.TestClock()
		connection := &federationdomain.IdentityProviderConnection{
			ID: pgfixtures.NewUUID(t), TenantID: tenantA.ID, DisplayName: "OIDC",
			Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
			Issuer: "https://idp.example", ClientID: "client", AuthorizationEndpoint: "https://idp.example/auth",
			TokenEndpoint: "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
			ClaimMapping: federationdomain.ClaimMapping{Subject: "sub", Username: "email"}, LinkingPolicy: federationdomain.LinkingNone,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := (&db_postgres.ConnectionRepository{Pool: db}).Save(context.Background(), connection); err != nil {
			t.Fatalf("seed connection: %v", err)
		}
		return testing_contract.Fixture{
			Attempts: &db_postgres.AttemptStore{Pool: db}, Replay: &db_postgres.ReplayStore{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			ProviderID: connection.ID, State: pgfixtures.UniqueID("state"), Now: now,
		}
	})
}
