package db_memory

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/tokenvault/internal/testing_contract"
)

//spec:covers REQ-TOKENVAULT-012: 版とリースの条件付き更新、REQ-TOKENVAULT-004: 連携の束縛と一回性、REQ-TOKENVAULT-002: 削除が接続を消すこと
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		store := NewStore()
		return testing_contract.Fixture{
			Connectors: store.Connectors(), Connections: store.Connections(), Attempts: store.Attempts(),
			TenantA: "tenant-a", TenantB: "tenant-b", UserA: "user-a", NewID: uuid.NewString,
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
