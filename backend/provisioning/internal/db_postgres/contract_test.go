package db_postgres_test

import (
	"testing"

	postgres "github.com/ambi/idmagic/backend/provisioning/internal/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

//spec:covers REQ-PROVISIONING-020: 保留はジョブとテナントを照合して保存され、再投入後の古いジョブでは変更できない。
//spec:covers REQ-PROVISIONING-021, REQ-PROVISIONING-015: 関連状態の削除と再登録は、別接続と別テナントを変えない。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		application := seedApplication(t, db, tenantA.ID)
		sibling := seedApplication(t, db, tenantA.ID)
		other := seedApplication(t, db, tenantB.ID)
		return testing_contract.Fixture{
			Connections: &postgres.ProvisioningConnectionRepository{Pool: db}, Tasks: &postgres.ProvisioningTaskRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			Links: &postgres.RemoteResourceLinkRepository{Pool: db}, SiblingApplication: sibling.ID, OtherApplication: other.ID,
			Application: application.ID, CredentialID: pgfixtures.NewUUID(t), Now: pgfixtures.TestClock(),
		}
	})
}
