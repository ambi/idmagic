package db_memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dbmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// TestAgentRevocationEpochRepository_AdvanceIsMonotonic — scenario
// `kill-switchは既発行トークンをintrospectionで即時無効化する` の前提: epoch は後退しない
// (fail-closed の単調増加保証)。
func TestAgentRevocationEpochRepository_AdvanceIsMonotonic(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewAgentRevocationEpochRepository()
	now := time.Now().UTC()

	first := ssdomain.AgentRevocationEpoch{
		AgentID: "agent_1", TenantID: "tenant-a",
		Epoch: now, Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}
	if err := repo.Advance(ctx, first); err != nil {
		t.Fatalf("first Advance: %v", err)
	}

	// 後退は拒否される。
	earlier := first
	earlier.Epoch = now.Add(-time.Hour)
	if err := repo.Advance(ctx, earlier); !errors.Is(err, ssdomain.ErrEpochNotAdvancing) {
		t.Fatalf("Advance backward = %v, want ErrEpochNotAdvancing", err)
	}
	got, err := repo.FindByAgent(ctx, "tenant-a", "agent_1")
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if !got.Epoch.Equal(now) {
		t.Fatalf("epoch regressed: got %v, want unchanged %v", got.Epoch, now)
	}

	// 前進は受理される。
	later := first
	later.Epoch = now.Add(time.Hour)
	later.Reason = ssdomain.RevocationReasonOwnerDisabled
	if err := repo.Advance(ctx, later); err != nil {
		t.Fatalf("Advance forward: %v", err)
	}
	got, err = repo.FindByAgent(ctx, "tenant-a", "agent_1")
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if !got.Epoch.Equal(now.Add(time.Hour)) || got.Reason != ssdomain.RevocationReasonOwnerDisabled {
		t.Fatalf("epoch did not advance correctly: %+v", got)
	}
}

func TestAgentRevocationEpochRepository_FindByAgentUnknownReturnsNil(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewAgentRevocationEpochRepository()
	got, err := repo.FindByAgent(ctx, "tenant-a", "no-such-agent")
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for unfailed agent, got %+v", got)
	}
}

func TestAgentRevocationEpochRepository_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	repo := dbmemory.NewAgentRevocationEpochRepository()
	now := time.Now().UTC()
	if err := repo.Advance(ctx, ssdomain.AgentRevocationEpoch{
		AgentID: "agent_1", TenantID: "tenant-a", Epoch: now, Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}); err != nil {
		t.Fatalf("Advance: %v", err)
	}
	got, err := repo.FindByAgent(ctx, "tenant-b", "agent_1")
	if err != nil {
		t.Fatalf("FindByAgent: %v", err)
	}
	if got != nil {
		t.Fatalf("cross-tenant lookup must return nil, got %+v", got)
	}
}
