package db_memory

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement/internal/testing_contract"
)

//spec:covers REQ-TENANCY-048: 確定した成果物だけの容量を、テナント別にメタデータから計測する。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Artifacts: NewCSVArtifactStore(), TenantA: "tenant-a", TenantB: "tenant-b"}
	})
}
