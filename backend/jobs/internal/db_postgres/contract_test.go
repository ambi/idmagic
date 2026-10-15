package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/jobs/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &JobRepository{Pool: db}, TenantID: tenant.ID, Now: pgtest.Now()}
	})
}
