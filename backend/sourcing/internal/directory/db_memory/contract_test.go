package db_memory

import (
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/testing_contract"
)

//spec:covers REQ-SOURCING-008 REQ-SOURCING-013
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: NewConnectorRepository(), TenantID: "tenant-a", OtherTenantID: "tenant-b"}
	})
}
