package db_postgres

import (
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/testing_contract"
)

//spec:covers REQ-SOURCING-008 REQ-SOURCING-013
func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &ConnectorRepository{Pool: db}, TenantID: tenant.ID, OtherTenantID: other.ID}
	})
}
