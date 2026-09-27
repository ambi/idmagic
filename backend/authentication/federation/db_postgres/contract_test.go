package db_postgres_test

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/authentication/federation/db_postgres"
	"github.com/ambi/idmagic/backend/authentication/federation/domain"
	"github.com/ambi/idmagic/backend/authentication/federation/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		now := pgfixtures.TestClock()
		connection := &domain.IdentityProviderConnection{
			ID: pgfixtures.NewUUID(t), TenantID: tenantA.ID, DisplayName: "OIDC",
			Protocol: domain.ProtocolOIDC, Status: domain.ConnectionActive,
			Issuer: "https://idp.example", ClientID: "client", AuthorizationEndpoint: "https://idp.example/auth",
			TokenEndpoint: "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
			ClaimMapping: domain.ClaimMapping{Subject: "sub", Username: "email"}, LinkingPolicy: domain.LinkingNone,
			CreatedAt: now, UpdatedAt: now,
		}
		if err := (&db_postgres.ConnectionRepository{Pool: db}).Save(context.Background(), connection); err != nil {
			t.Fatalf("seed connection: %v", err)
		}
		return testing_contract.Fixture{
			Attempts: &db_postgres.AttemptStore{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			ProviderID: connection.ID, State: pgfixtures.UniqueID("state"), Now: now,
		}
	})
}
