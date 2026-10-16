package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/authentication/password/testing_contract"
	userpg "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	user := pgfixtures.SeedUser(t, db, tenant.ID)
	history := &PasswordHistoryRepository{Pool: db}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{History: history, Reset: NewPasswordResetTokenStore(db, userpg.UserWriterInTx), User: user, Now: pgfixtures.TestClock()}
	})
}
