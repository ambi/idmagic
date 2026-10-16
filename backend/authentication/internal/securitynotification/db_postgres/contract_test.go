package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement"

	"github.com/ambi/idmagic/backend/authentication/internal/securitynotification/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		user := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db, nil).UserRepo, tenant.ID)
		other := pgfixtures.SeedUser(t, idmanagement.NewPostgresModule(db, nil).UserRepo, tenant.ID)
		return testing_contract.Fixture{
			Preferences: &PreferenceRepository{Pool: db}, Devices: &KnownDeviceRepository{Pool: db},
			UserID: user.ID, OtherUserID: other.ID, Now: pgtest.Now(),
		}
	})
}
