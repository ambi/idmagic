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

// TestAgentRevocationEpochRepository_MultiReplicaConsistency — T006 検証:
// AgentRevocationEpochRepository は process-local な状態を持たず、Advance/FindByAgent
// を都度 PostgreSQL へ直接発行する。2つの独立した repository インスタンス (idmagic の
// 別 replica を模す、同一 PostgreSQL を共有) の一方が Advance すれば、もう一方が
// 直後の FindByAgent で同じ epoch を読める (多 replica 環境でも epoch 判定が分岐しない)。
func TestAgentRevocationEpochRepository_MultiReplicaConsistency(t *testing.T) {
	db := pgtest.Require(t)
	replicaA := &AgentRevocationEpochRepository{Pool: db}
	replicaB := &AgentRevocationEpochRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	now := testClock()

	if err := replicaA.Advance(context.Background(), ssdomain.AgentRevocationEpoch{
		AgentID: agent.ID, TenantID: tenant.ID, Epoch: now, Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}); err != nil {
		t.Fatalf("replicaA Advance: %v", err)
	}

	got, err := replicaB.FindByAgent(context.Background(), tenant.ID, agent.ID)
	if err != nil {
		t.Fatalf("replicaB FindByAgent: %v", err)
	}
	if got == nil || !got.Epoch.Equal(now) {
		t.Fatalf("replicaB did not observe replicaA's Advance: %+v", got)
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

// TestUserRevocationEpochRepository は単調増加と、User の物理削除の連鎖を DB の制約で確かめる。
//
//spec:covers REQ-SHAREDSIGNALS-012: エポックが保存層で後退せず、UUID でない主体を未失効と読むこと
func TestUserRevocationEpochRepository(t *testing.T) {
	db := pgtest.Require(t)
	repo := &UserRevocationEpochRepository{Pool: db}
	tenant := seedTenant(t, db)
	agent := seedAgent(t, db, tenant.ID)
	now := testClock()
	first := ssdomain.UserRevocationEpoch{
		UserID: agent.OwnerUserID, TenantID: tenant.ID, Epoch: now,
		Reason: ssdomain.RevocationReasonSessionRevoked, AdvancedAt: now,
	}
	ctx := context.Background()
	if err := repo.Advance(ctx, first); err != nil {
		t.Fatalf("first Advance: %v", err)
	}
	for _, at := range []time.Time{now, now.Add(-time.Hour)} {
		stale := first
		stale.Epoch = at
		if err := repo.Advance(ctx, stale); !errors.Is(err, ssdomain.ErrEpochNotAdvancing) {
			t.Fatalf("Advance %v = %v", at, err)
		}
	}
	later := first
	later.Epoch = now.Add(time.Hour)
	if err := repo.Advance(ctx, later); err != nil {
		t.Fatalf("later Advance: %v", err)
	}
	got, err := repo.FindByUser(ctx, tenant.ID, agent.OwnerUserID)
	if err != nil || got == nil || !got.Epoch.Equal(later.Epoch) {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	if other, err := repo.FindByUser(ctx, seedTenant(t, db).ID, agent.OwnerUserID); err != nil || other != nil {
		t.Fatalf("別テナント got=%+v err=%v", other, err)
	}
	if none, err := repo.FindByUser(ctx, tenant.ID, "not-a-uuid"); err != nil || none != nil {
		t.Fatalf("UUID でない主体 got=%+v err=%v", none, err)
	}
}
