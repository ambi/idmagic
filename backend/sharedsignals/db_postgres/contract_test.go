package db_postgres

import (
	"testing"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/sharedsignals/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := seedTenant(t, db)
		tenantB := seedTenant(t, db)
		agent := seedAgent(t, db, tenantA.ID)
		return testing_contract.Fixture{
			Epochs: &AgentRevocationEpochRepository{Pool: db}, TenantA: tenantA.ID, TenantB: tenantB.ID,
			AgentID: agent.ID, Now: testClock(),
		}
	})
}
