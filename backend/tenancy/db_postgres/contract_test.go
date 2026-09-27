package db_postgres

import (
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	now := pgtest.Now()
	tenant := &domain.Tenant{ID: "44444444-4444-4444-4444-444444444441", Realm: "contract-realm", DisplayName: "Contract Tenant", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &TenantRepository{Pool: db}, Tenant: tenant}
	})
}
