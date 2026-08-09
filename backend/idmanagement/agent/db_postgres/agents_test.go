package db_postgres

import (
	"context"
	"testing"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

func TestAgentRepositoryRoundTripAndBindings(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	owner := seedUser(t, db, tenant.ID)
	client := seedClient(t, db, tenant.ID)
	repo := &AgentRepository{Pool: db}
	ctx := context.Background()

	now := testClock()
	agent := &agentdomain.Agent{
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

	list, err := repo.ListAll(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list agents: %v len=%d", err, len(list))
	}

	added, err := repo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
		AgentID: agent.ID, ClientID: client.ClientID, CreatedAt: now,
	})
	if err != nil || !added {
		t.Fatalf("add binding: %v added=%v", err, added)
	}
	// 冪等: 同じ束縛の再追加は false。
	added, err = repo.AddBinding(ctx, &agentdomain.AgentCredentialBinding{
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

func TestAgentRepositoryListPage(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	owner := seedUser(t, db, tenant.ID)
	repo := &AgentRepository{Pool: db}
	ctx := context.Background()
	now := testClock()

	for _, name := range []string{"Charlie", "Alpha", "Bravo", "Delta", "Echo"} {
		a := &agentdomain.Agent{
			ID: newUUID(t), TenantID: tenant.ID, Name: name, Kind: idmdomain.AgentKindAutonomous,
			OwnerUserID: owner.ID, Status: idmdomain.AgentStatusActive, Roles: []string{},
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Save(ctx, a); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	first, err := repo.ListPage(ctx, tenant.ID, "", "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].Name != "Alpha" || first[1].Name != "Bravo" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, tenant.ID, last.Name, last.ID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].Name != "Charlie" || next[1].Name != "Delta" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
