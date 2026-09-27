package db_memory_test

import (
	"testing"
	"time"

	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	"github.com/ambi/idmagic/backend/sharedsignals/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Epochs: dbmemory.NewAgentRevocationEpochRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			AgentID: "agent-1", Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
