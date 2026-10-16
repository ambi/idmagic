package db_postgres_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/internal/client/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/client/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	client := pgfixtures.SeedClient(t, &db_postgres.OAuth2ClientRepository{Pool: db}, pgfixtures.SeedTenant(t, db).ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &db_postgres.OAuth2ClientRepository{Pool: db}, Client: client}
	})
}
