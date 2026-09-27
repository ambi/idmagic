package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{
			Repository: NewTenantRepository(),
			Tenant:     &domain.Tenant{ID: "tenant-contract", Realm: "contract-realm", DisplayName: "Contract Tenant", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		}
	})
}
