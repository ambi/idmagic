package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/wsfederation/internal/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{
			Repository: NewWsFedRelyingPartyRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}

//spec:covers REQ-WSFEDERATION-001: 観測記録がテナントとドメインごとに一件で、古い取得時刻で上書きされず、別テナントから読めないこと。
func TestEntraObservationPersistenceContract(t *testing.T) {
	testing_contract.RunEntraObservations(t, func(*testing.T) testing_contract.ObservationFixture {
		return testing_contract.ObservationFixture{
			Repository: NewEntraFederationObservationRepository(), TenantA: "tenant-a", TenantB: "tenant-b",
			Now: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		}
	})
}
