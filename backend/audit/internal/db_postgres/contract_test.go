package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/audit/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

//spec:covers REQ-AUDIT-013: PostgreSQL の保持期間の削除が、種類別の境界を持つ種類をデフォルトの境界で消さず、境界の外側だけを全テナントから消す。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			Repository: &AuditEventRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID, Now: pgtest.Now(),
		}
	})
}
