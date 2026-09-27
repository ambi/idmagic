package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: NewTrustedDeviceRepository(), TenantID: "tenant-a", UserID: "user-1", Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	})
}
