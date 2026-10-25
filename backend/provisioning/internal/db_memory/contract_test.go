package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/internal/testing_contract"
)

//spec:covers REQ-PROVISIONING-020: 保留はジョブとテナントを照合して保存され、再投入後の古いジョブでは変更できない。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Connections: NewProvisioningConnectionRepository(), Tasks: NewProvisioningTaskRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			Application: "application-1", CredentialID: "credential-1",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
