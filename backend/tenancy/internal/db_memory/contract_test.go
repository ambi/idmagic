package db_memory

import (
	"strconv"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/internal/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{
			Repository: NewTenantRepository(),
			Tenant:     &domain.Tenant{ID: "tenant-contract", Realm: "contract-realm", DisplayName: "Contract Tenant", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now},
		}
	})
}

//spec:covers EX-TENANCY-036-01, EX-TENANCY-036-02, EX-TENANCY-036-03, EX-TENANCY-036-04, EX-TENANCY-037-01, EX-TENANCY-037-02: メモリーの実装が、上書きとデフォルトの実効値、拒否した加算の使用量、未知のリソース、0 で止まる減算、全置換の更新、使用量を下回る上限を共通の契約どおりに扱う。
func TestQuotaContract(t *testing.T) {
	testing_contract.RunQuota(t, func(*testing.T) testing_contract.QuotaFixture {
		return testing_contract.QuotaFixture{Repository: NewQuotaRepository(), TenantID: "tenant-quota"}
	})
}

//spec:covers REQ-TENANCY-047: Soft の上限を超える加算、0 の上書き、2 GiB 超、負の量の拒否、並行の計数をメモリで確かめる。
func TestSoftQuotaContract(t *testing.T) {
	repository := NewQuotaRepository()
	var next int
	testing_contract.RunSoftQuota(t, func(*testing.T) testing_contract.QuotaFixture {
		next++
		return testing_contract.QuotaFixture{Repository: repository, TenantID: "soft-quota-" + strconv.Itoa(next)}
	})
}
