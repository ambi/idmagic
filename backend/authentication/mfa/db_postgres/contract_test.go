package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/authentication/mfa/testing_contract"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		user := pgfixtures.SeedUser(t, db, tenantA.ID)
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return testing_contract.Fixture{
			Repository: &MfaEnrollmentBypassRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			UserID: user.ID, IssuedBy: user.ID, BypassID: id, Now: pgfixtures.TestClock(),
		}
	})
}
