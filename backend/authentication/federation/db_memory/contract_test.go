package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/federation/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		repositories := NewRepositories()
		return testing_contract.Fixture{
			Attempts: repositories.Attempts, TenantA: "tenant-a", TenantB: "tenant-b",
			ProviderID: "provider-1", State: "state-1",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
