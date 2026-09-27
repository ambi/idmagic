package db_postgres

import (
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	"github.com/ambi/idmagic/backend/idmanagement/agent/testing_contract"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestPersistenceContract(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	owner := seedUser(t, db, tenant.ID)
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := testClock()
		return testing_contract.Fixture{Repository: &AgentRepository{Pool: db}, TenantID: tenant.ID, Agent: &domain.Agent{ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("contract-agent"), Kind: idmdomain.AgentKindAutonomous, OwnerUserID: owner.ID, Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}}
	})
}
