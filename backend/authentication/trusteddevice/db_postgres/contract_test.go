package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/testing_contract"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenant := pgfixtures.SeedTenant(t, db)
		user := pgfixtures.SeedUser(t, db, tenant.ID)
		return testing_contract.Fixture{Repository: &TrustedDeviceRepository{Pool: db}, TenantID: tenant.ID, UserID: user.ID, Now: pgtest.Now()}
	})
}
