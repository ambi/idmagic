package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: NewDataKeyRepository(), TenantID: "tenant-contract", Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	})
}
