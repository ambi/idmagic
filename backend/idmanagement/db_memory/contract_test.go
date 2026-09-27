package db_memory

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Artifacts: NewCSVArtifactStore(), TenantA: "tenant-a", TenantB: "tenant-b"}
	})
}
