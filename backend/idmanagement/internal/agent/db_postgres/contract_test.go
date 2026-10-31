package db_postgres

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	"github.com/ambi/idmagic/backend/idmanagement/internal/agent/testing_contract"
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

//spec:covers REQ-IDMANAGEMENT-097: 再送と順序逆転は値と決定を変えない
//spec:covers REQ-IDMANAGEMENT-096: テナントの上限を超えるバッチは一件も保存しない
//spec:covers REQ-IDMANAGEMENT-102: 期限を過ぎた未確認と除外を消し、確認を残す
func TestAgentObservationContract(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunAgentObservationContract(t, func(t *testing.T) testing_contract.ObservationFixture {
		t.Helper()
		tenant := seedTenant(t, db)
		other := seedTenant(t, db)
		owner := seedUser(t, db, tenant.ID)
		now := testClock()
		agents := &AgentRepository{Pool: db}
		agent := &domain.Agent{ID: newUUID(t), TenantID: tenant.ID, Name: uniqueID("contract-agent"), Kind: idmdomain.AgentKindAutonomous, OwnerUserID: owner.ID, Status: idmdomain.AgentStatusActive, Roles: []string{}, CreatedAt: now, UpdatedAt: now}
		if err := agents.Save(context.Background(), agent); err != nil {
			t.Fatal(err)
		}
		return testing_contract.ObservationFixture{
			Repository: &AgentObservationRepository{Pool: db}, Agents: agents, TenantID: tenant.ID, OtherTenantID: other.ID,
			Agent: agent, MissingAgent: newUUID(t), NewID: func() string { return newUUID(t) },
		}
	})
}

//spec:covers REQ-IDMANAGEMENT-098: 外部の登録 ID と確認先の id が UUID でなくても、存在しないものとして扱う
func TestAgentLookupsTreatNonUUIDAsAbsent(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	got, err := (&AgentRepository{Pool: db}).FindByClientID(context.Background(), tenant.ID, "not-a-uuid")
	if err != nil || got != nil {
		t.Fatalf("FindByClientID=(%+v,%v)", got, err)
	}
	if got, err := (&AgentRepository{Pool: db}).FindByID(context.Background(), tenant.ID, "not-a-uuid"); err != nil || got != nil {
		t.Fatalf("FindByID=(%+v,%v)", got, err)
	}
}
