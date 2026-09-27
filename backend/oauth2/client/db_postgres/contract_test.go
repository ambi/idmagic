package db_postgres_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/client/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/client/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	client := pgfixtures.SeedClient(t, db, pgfixtures.SeedTenant(t, db).ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &db_postgres.OAuth2ClientRepository{Pool: db}, Client: client}
	})
}
