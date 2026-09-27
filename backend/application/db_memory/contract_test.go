package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Categories: NewApplicationCategoryRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			CategoryA: "category-a", CategoryB: "category-b",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
