package db_postgres_test

import (
	"testing"

	postgres "github.com/ambi/idmagic/backend/provisioning/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		application := seedApplication(t, db, tenantA.ID)
		return testing_contract.Fixture{
			Connections: &postgres.ProvisioningConnectionRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			Application: application.ID, CredentialID: pgfixtures.NewUUID(t), Now: pgfixtures.TestClock(),
		}
	})
}
