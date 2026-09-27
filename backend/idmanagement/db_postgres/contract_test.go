package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{Artifacts: &CSVArtifactStore{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID}
	})
}
