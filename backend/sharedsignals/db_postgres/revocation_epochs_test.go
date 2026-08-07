package db_postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// TestAgentRevocationEpochRepository_AdvanceIsMonotonic — scenario
// `kill-switchは既発行トークンをintrospectionで即時無効化する` の前提: DB 制約レベルで
// epoch の後退を拒否する (fail-closed の単調増加保証、ON CONFLICT ... WHERE)。
func TestAgentRevocationEpochRepository_AdvanceIsMonotonic(t *testing.T) {
	db := pgtest.Require(t)
	repo := &AgentRevocationEpochRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	now := testClock()

	first := ssdomain.AgentRevocationEpoch{
		AgentID: agent.ID, TenantID: tenant.ID, Epoch: now,
		Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}
	if err := repo.Advance(context.Background(), first); err != nil {
		t.Fatalf("first Advance: %v", err)
	}

	earlier := first
	earlier.Epoch = now.Add(-time.Hour)
	if err := repo.Advance(context.Background(), earlier); !errors.Is(err, ssdomain.ErrEpochNotAdvancing) {
		t.Fatalf("Advance backward = %v, want ErrEpochNotAdvancing", err)
	}
	got, err := repo.FindByAgent(context.Background(), tenant.ID, agent.ID)
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if !got.Epoch.Equal(now) {
		t.Fatalf("epoch regressed: got %v, want unchanged %v", got.Epoch, now)
	}

	later := first
	later.Epoch = now.Add(time.Hour)
	later.Reason = ssdomain.RevocationReasonOwnerDisabled
	if err := repo.Advance(context.Background(), later); err != nil {
		t.Fatalf("Advance forward: %v", err)
	}
	got, err = repo.FindByAgent(context.Background(), tenant.ID, agent.ID)
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if !got.Epoch.Equal(now.Add(time.Hour)) || got.Reason != ssdomain.RevocationReasonOwnerDisabled {
		t.Fatalf("epoch did not advance correctly: %+v", got)
	}
}

func TestAgentRevocationEpochRepository_FindByAgentUnknownReturnsNil(t *testing.T) {
	db := pgtest.Require(t)
	repo := &AgentRevocationEpochRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)

	got, err := repo.FindByAgent(context.Background(), tenant.ID, agent.ID)
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for unfailed agent, got %+v", got)
	}
}

// TestAgentRevocationEpochRepository_CascadesOnAgentDelete は
// agent_revocation_epochs.agent_id の ON DELETE CASCADE を検証する: 削除された
// Agent の epoch は Agent と一緒に消える (epoch が孤立レコードとして残らない)。
func TestAgentRevocationEpochRepository_CascadesOnAgentDelete(t *testing.T) {
	db := pgtest.Require(t)
	repo := &AgentRevocationEpochRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	now := testClock()

	if err := repo.Advance(context.Background(), ssdomain.AgentRevocationEpoch{
		AgentID: agent.ID, TenantID: tenant.ID, Epoch: now, Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	if _, err := db.Exec(context.Background(), "DELETE FROM agents WHERE id=$1", agent.ID); err != nil {
		t.Fatalf("delete agent: %v", err)
	}
	got, err := repo.FindByAgent(context.Background(), tenant.ID, agent.ID)
	if err != nil {
		t.Fatalf("FindByAgent after cascade: %v", err)
	}
	if got != nil {
		t.Fatalf("expected epoch to be cascade-deleted with agent, got %+v", got)
	}
}
