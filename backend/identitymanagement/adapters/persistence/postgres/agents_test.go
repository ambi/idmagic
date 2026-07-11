package postgres

import (
	"context"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

func TestAgentRepositoryRoundTripAndBindings(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	owner := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	repo := &AgentRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	agent := &idmdomain.Agent{
		ID:          newUUID(t),
		TenantID:    tenant.ID,
		Name:        "svc-agent",
		Kind:        idmdomain.AgentKindAutonomous,
		OwnerUserID: owner.ID,
		Status:      idmdomain.AgentStatusActive,
		Roles:       []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Save(ctx, agent); err != nil {
		t.Fatalf("save agent: %v", err)
	}

	got, err := repo.FindByID(ctx, tenant.ID, agent.ID)
	if err != nil || got == nil || got.OwnerUserID != owner.ID {
		t.Fatalf("find agent: %v %+v", err, got)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list agents: %v len=%d", err, len(list))
	}

	added, err := repo.AddBinding(ctx, &idmdomain.AgentCredentialBinding{
		AgentID: agent.ID, ClientID: client.ClientID, CreatedAt: now,
	})
	if err != nil || !added {
		t.Fatalf("add binding: %v added=%v", err, added)
	}
	// 冪等: 同じ束縛の再追加は false。
	added, err = repo.AddBinding(ctx, &idmdomain.AgentCredentialBinding{
		AgentID: agent.ID, ClientID: client.ClientID, CreatedAt: now,
	})
	if err != nil || added {
		t.Fatalf("duplicate binding should be false: %v added=%v", err, added)
	}

	bindings, err := repo.ListBindings(ctx, tenant.ID, agent.ID)
	if err != nil || len(bindings) != 1 {
		t.Fatalf("list bindings: %v len=%d", err, len(bindings))
	}

	byClient, err := repo.FindByClientID(ctx, tenant.ID, client.ClientID)
	if err != nil || byClient == nil || byClient.ID != agent.ID {
		t.Fatalf("find by client: %v %+v", err, byClient)
	}

	removed, err := repo.RemoveBinding(ctx, tenant.ID, agent.ID, client.ClientID)
	if err != nil || !removed {
		t.Fatalf("remove binding: %v removed=%v", err, removed)
	}

	if err := repo.Delete(ctx, tenant.ID, agent.ID); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
}
