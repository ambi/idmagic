package db_postgres

import (
	"os"
	"testing"

	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/wsfederation/testing_contract"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			Repository: &WsFedRelyingPartyRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			Now: pgtest.Now(),
		}
	})
}
