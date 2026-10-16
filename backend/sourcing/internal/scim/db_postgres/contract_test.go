package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/sourcing/internal/scim/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db, nil).UserRepo, tenant.ID)
	group := pgfixtures.SeedGroup(t, idmanagement.NewPostgresModule(db, nil).GroupRepo, tenant.ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &ScimRepository{Pool: db}, TenantID: tenant.ID, UserID: user.ID, GroupID: group.ID}
	})
}
