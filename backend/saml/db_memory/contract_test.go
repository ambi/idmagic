package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/saml/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Replay: NewAuthnRequestReplayStore(), TenantA: "tenant-a", TenantB: "tenant-b",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
