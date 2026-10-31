package db_postgres_test

import (
	"context"
	"testing"

	oauth2postgres "github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/ambi/idmagic/backend/oauth2/internal/testing_contract"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
)

func TestPersistenceContract(t *testing.T) {
	testing_contract.Run(t, func(t *testing.T) testing_contract.Fixture {
		t.Helper()
		db := pgtest.Require(t)
		tenantA := pgfixtures.SeedTenant(t, db)
		tenantB := pgfixtures.SeedTenant(t, db)
		return testing_contract.Fixture{
			DetailTypes: &oauth2postgres.AuthorizationDetailTypeRepository{Pool: db},
			TenantA:     tenantA.ID, TenantB: tenantB.ID, Now: pgfixtures.TestClock(),
		}
	})
}

//spec:covers REQ-OAUTH2-072: PostgreSQL の保存先が許可の共有契約を満たすこと
func TestCrossAppAccessPolicyRepositoryContract(t *testing.T) {
	db := pgtest.Require(t)
	testing_contract.RunCrossAppAccess(t, testing_contract.CrossAppAccessFixture{
		Policies: &oauth2postgres.CrossAppAccessPolicyRepository{Pool: db},
		TenantA:  seedTenant(t, db).ID, TenantB: seedTenant(t, db).ID,
		Now: testClock(),
	})
}

//spec:covers REQ-OAUTH2-077: PostgreSQL の保存先がガードレールの共有契約を満たし、Agent の削除に追随すること
func TestAgentGuardrailPolicyRepositoryContract(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenant := seedTenant(t, db)
	owner := seedUser(t, db, tenant.ID)
	agentID := newUUID(t)
	if _, err := db.Exec(ctx, `INSERT INTO agents (id, tenant_id, name, kind, owner_user_id, name_key) VALUES ($1, $2, $3, 'autonomous', $4, $3)`,
		agentID, tenant.ID, "guardrail-"+agentID, owner.ID); err != nil {
		t.Fatal(err)
	}
	repo := &oauth2postgres.AgentGuardrailPolicyRepository{Pool: db}
	testing_contract.RunAgentGuardrail(t, testing_contract.AgentGuardrailFixture{
		Repo: repo, Context: ctx, TenantID: tenant.ID, Other: seedTenant(t, db).ID, AgentID: agentID, Now: testClock(),
	})
	if _, err := db.Exec(ctx, `DELETE FROM agents WHERE id = $1`, agentID); err != nil {
		t.Fatal(err)
	}
	if found, err := repo.Find(ctx, tenant.ID, agentID); err != nil || found != nil {
		t.Fatalf("policy must follow the agent deletion: (%+v, %v)", found, err)
	}
}
