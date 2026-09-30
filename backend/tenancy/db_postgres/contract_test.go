package db_postgres

import (
	"context"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	now := pgtest.Now()
	tenant := &domain.Tenant{ID: "44444444-4444-4444-4444-444444444441", Realm: "contract-realm", DisplayName: "Contract Tenant", Status: domain.TenantStatusActive, CreatedAt: now, UpdatedAt: now}
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		return testing_contract.Fixture{Repository: &TenantRepository{Pool: db}, Tenant: tenant}
	})
}

//spec:covers EX-TENANCY-036-01, EX-TENANCY-036-02, EX-TENANCY-036-03, EX-TENANCY-036-04, EX-TENANCY-037-01, EX-TENANCY-037-02: PostgreSQL の実装が、上書きとデフォルトの実効値、拒否した加算の使用量、未知のリソース、0 で止まる減算、全置換の更新、使用量を下回る上限を共通の契約どおりに扱う。
func TestQuotaContract(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunQuota(t, func(t *testing.T) testing_contract.QuotaFixture {
		t.Helper()
		tenant := seedQuotaTenant(context.Background(), t, db)
		return testing_contract.QuotaFixture{Repository: NewQuotaRepository(db), TenantID: tenant.ID}
	})
}
