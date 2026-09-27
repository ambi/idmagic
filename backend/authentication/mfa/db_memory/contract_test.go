package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/mfa/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Repository: NewMfaEnrollmentBypassRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			UserID: "user-1", IssuedBy: "admin-1", BypassID: "bypass-1",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
