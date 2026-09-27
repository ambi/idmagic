package db_postgres

import (
	"testing"

	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/workloadidentity/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := seedTenant(t, db)
		tenantB := seedTenant(t, db)
		return testing_contract.Fixture{
			TrustBundles: &WorkloadTrustBundleRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			BundleID: pgfixtures.NewUUID(t), OtherID: pgfixtures.NewUUID(t), Now: testClock(),
		}
	})
}
