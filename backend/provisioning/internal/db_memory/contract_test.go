package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/internal/testing_contract"
)

//spec:covers REQ-PROVISIONING-020: 保留はジョブとテナントを照合して保存され、再投入後の古いジョブでは変更できない。
//spec:covers REQ-PROVISIONING-021, REQ-PROVISIONING-015: 関連状態の削除と再登録は、別接続と別テナントを変えない。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		tasks := NewProvisioningTaskRepository()
		links := NewRemoteResourceLinkRepository()
		return testing_contract.Fixture{
			Connections: NewProvisioningConnectionRepository(tasks, links), Tasks: tasks, TenantA: "tenant-a", TenantB: "tenant-b",
			Links: links, SiblingApplication: "application-2", OtherApplication: "application-3",
			Application: "application-1", CredentialID: "credential-1",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
