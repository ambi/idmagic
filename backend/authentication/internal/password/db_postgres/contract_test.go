package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/authentication/internal/password/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	users := idmanagement.NewPostgresModule(db).UserRepo
	user := pgfixtures.SeedUser(t, users, tenant.ID)
	history := &PasswordHistoryRepository{Pool: db}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{History: history, Reset: NewPasswordResetTokenStore(db, idmanagement.UserWriterInTx), Users: users, User: user, Now: pgfixtures.TestClock()}
	})
}
