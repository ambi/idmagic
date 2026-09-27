package db_postgres

import (
	"testing"

	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/sourcing/scim/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	group := pgfixtures.SeedGroup(t, db, tenant.ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &ScimRepository{Pool: db}, TenantID: tenant.ID, UserID: user.ID, GroupID: group.ID}
	})
}
