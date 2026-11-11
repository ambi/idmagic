package db_postgres

import (
	"os"
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	"github.com/ambi/idmagic/backend/wsfederation/internal/testing_contract"
)

func TestMain(m *testing.M) { os.Exit(pgtest.Main(m)) }

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			Repository: &WsFedRelyingPartyRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			Now: pgtest.Now(),
		}
	})
}

//spec:covers REQ-WSFEDERATION-001: 観測記録がテナントとドメインごとに一件で、条件付きの更新が古い取得時刻で上書きせず、別テナントから読めないこと。
func TestEntraObservationPersistenceContract(t *testing.T) {
	testing_contract.RunEntraObservations(t, func(t *testing.T) testing_contract.ObservationFixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.ObservationFixture{
			Repository: &EntraFederationObservationRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			Now: pgtest.Now(),
		}
	})
}
