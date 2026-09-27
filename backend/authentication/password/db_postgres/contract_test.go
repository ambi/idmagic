package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/authentication/password/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	history := &PasswordHistoryRepository{Pool: db}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{History: history, Reset: &PasswordResetTokenStore{Pool: db}, User: user, Now: pgfixtures.TestClock()}
	})
}
