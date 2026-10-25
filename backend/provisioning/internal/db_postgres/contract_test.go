package db_postgres_test

import (
	"testing"

	postgres "github.com/ambi/idmagic/backend/provisioning/internal/db_postgres"
	"github.com/ambi/idmagic/backend/provisioning/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

//spec:covers REQ-PROVISIONING-020: 保留はジョブとテナントを照合して保存され、再投入後の古いジョブでは変更できない。
func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		application := seedApplication(t, db, tenantA.ID)
		return testing_contract.Fixture{
			Connections: &postgres.ProvisioningConnectionRepository{Pool: db}, Tasks: &postgres.ProvisioningTaskRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			Application: application.ID, CredentialID: pgfixtures.NewUUID(t), Now: pgfixtures.TestClock(),
		}
	})
}
