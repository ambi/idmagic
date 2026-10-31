package db_memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/idmanagement/internal/agent/testing_contract"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(*testing.T) testing_contract.Fixture {
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		return testing_contract.Fixture{Repository: NewAgentRepository(), TenantID: "tenant-contract", Agent: &domain.Agent{ID: "agent-contract", TenantID: "tenant-contract", Name: "contract-agent", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "user-contract", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}}
	})
}

//spec:covers REQ-IDMANAGEMENT-097: 再送と順序逆転は値と決定を変えない
//spec:covers REQ-IDMANAGEMENT-102: 期限を過ぎた未確認と除外を消し、確認を残す
func TestAgentObservationContract(t *testing.T) {
	sequence := 0
	testing_contract.RunAgentObservationContract(t, func(t *testing.T) testing_contract.ObservationFixture {
		t.Helper()
		agents := NewAgentRepository()
		now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
		agent := &domain.Agent{ID: "agent-contract", TenantID: "tenant-contract", Name: "contract-agent", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "user-contract", Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}
		if err := agents.Save(context.Background(), agent); err != nil {
			t.Fatal(err)
		}
		return testing_contract.ObservationFixture{
			Repository: NewAgentObservationRepository(agents), Agents: agents, TenantID: "tenant-contract", OtherTenantID: "tenant-other",
			Agent: agent, MissingAgent: "agent-missing",
			NewID: func() string { sequence++; return fmt.Sprintf("observation-%d", sequence) },
		}
	})
}
