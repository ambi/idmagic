package db_memory

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	"github.com/ambi/idmagic/backend/idmanagement/agent/testing_contract"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{Repository: NewAgentRepository(), TenantID: "tenant-contract", Agent: &domain.Agent{ID: "agent-contract", TenantID: "tenant-contract", Name: "contract-agent", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "user-contract", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}}
	})
}
