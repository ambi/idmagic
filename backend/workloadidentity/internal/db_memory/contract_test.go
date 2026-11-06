package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/workloadidentity/internal/testing_contract"
)

//spec:covers REQ-WORKLOADIDENTITY-008
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			TrustBundles: NewWorkloadTrustBundleRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			BundleID: "bundle-1", OtherID: "bundle-2", SPIFFEIDs: [3]string{"bundle-3", "bundle-4", "bundle-5"},
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
