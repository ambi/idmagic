package db_memory

import (
	"testing"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/testing_contract"
)

//spec:covers REQ-SOURCING-008 REQ-SOURCING-013
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: NewConnectorRepository(), TenantID: "tenant-a", OtherTenantID: "tenant-b"}
	})
}

//spec:covers REQ-SOURCING-015, REQ-SOURCING-017, REQ-SOURCING-018: メモリの保存先が、対応の範囲、実行の打ち切り、バッチの一度だけの記録と件数、完了の条件を守る。
func TestIngestionPersistenceContract(t *testing.T) {
	newID := func(*testing.T, string) string { return uuid.NewString() }
	testing_contract.RunIngestion(t, testing_contract.IngestionFixture{
		Correlations: NewCorrelationRepository(), Runs: NewIngestionRunRepository(),
		TenantID: "tenant-a", OtherTenantID: "tenant-b", NewConnector: newID, NewUser: newID,
	})
}
