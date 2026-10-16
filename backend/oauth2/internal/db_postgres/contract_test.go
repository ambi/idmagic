package db_postgres_test

import (
	"testing"

	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			DetailTypes: &oauth2postgres.AuthorizationDetailTypeRepository{Pool: db},
			TenantA:     tenantA.ID, TenantB: tenantB.ID, Now: pgfixtures.TestClock(),
		}
	})
}
