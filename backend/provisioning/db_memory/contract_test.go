package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Connections: NewProvisioningConnectionRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			Application: "application-1", CredentialID: "credential-1",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
