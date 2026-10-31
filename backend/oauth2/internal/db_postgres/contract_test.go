package db_postgres_test

import (
	"testing"

	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			DetailTypes: &oauth2postgres.AuthorizationDetailTypeRepository{Pool: db},
			TenantA:     tenantA.ID, TenantB: tenantB.ID, Now: pgfixtures.TestClock(),
		}
	})
}

//spec:covers REQ-OAUTH2-072: PostgreSQL の保存先が許可の共有契約を満たすこと
func TestCrossAppAccessPolicyRepositoryContract(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunCrossAppAccess(t, testing_contract.CrossAppAccessFixture{
		Policies: &oauth2postgres.CrossAppAccessPolicyRepository{Pool: db},
		TenantA:  seedTenant(t, db).ID, TenantB: seedTenant(t, db).ID,
		Now: testClock(),
	})
}
