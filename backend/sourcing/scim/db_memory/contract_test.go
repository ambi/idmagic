package db_memory

import (
	"testing"

	"github.com/ambi/idmagic/backend/sourcing/scim/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: NewScimRepository(), TenantID: "tenant-contract", UserID: "user-contract", GroupID: "group-contract"}
	})
}
