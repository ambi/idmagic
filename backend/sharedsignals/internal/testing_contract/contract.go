// Package testing_contract defines the shared Shared Signals persistence contract.
package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/sharedsignals/ports"
)

type Fixture struct {
	Epochs  ports.AgentRevocationEpochRepository
	TenantA string
	TenantB string
	AgentID string
	Now     time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if missing, err := f.Epochs.FindByAgent(ctx, f.TenantA, f.AgentID); err != nil || missing != nil {
		t.Fatalf("missing FindByAgent = (%+v, %v)", missing, err)
	}
	first := domain.AgentRevocationEpoch{
		AgentID: f.AgentID, TenantID: f.TenantA, Epoch: f.Now,
		Reason: domain.RevocationReasonAgentKilled, AdvancedAt: f.Now,
	}
	if err := f.Epochs.Advance(ctx, first); err != nil {
		t.Fatalf("first Advance: %v", err)
	}
	if leaked, err := f.Epochs.FindByAgent(ctx, f.TenantB, f.AgentID); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByAgent = (%+v, %v)", leaked, err)
	}
	earlier := first
	earlier.Epoch = f.Now.Add(-time.Hour)
	if err := f.Epochs.Advance(ctx, earlier); !errors.Is(err, domain.ErrEpochNotAdvancing) {
		t.Fatalf("backward Advance = %v", err)
	}
	unchanged, err := f.Epochs.FindByAgent(ctx, f.TenantA, f.AgentID)
	if err != nil || unchanged == nil || !unchanged.Epoch.Equal(f.Now) {
		t.Fatalf("FindByAgent after backward Advance = (%+v, %v)", unchanged, err)
	}
	// 同じ時刻への前進は進めない。所有者の停止と、それに伴う Agent の無効化が同じ時刻に
	// 重なっても、失効を二重に記録しないためである。
	sameTime := first
	sameTime.Reason = domain.RevocationReasonAgentDisabled
	if err := f.Epochs.Advance(ctx, sameTime); !errors.Is(err, domain.ErrEpochNotAdvancing) {
		t.Fatalf("same-time Advance = %v", err)
	}
	if kept, err := f.Epochs.FindByAgent(ctx, f.TenantA, f.AgentID); err != nil || kept == nil || kept.Reason != first.Reason {
		t.Fatalf("FindByAgent after same-time Advance = (%+v, %v)", kept, err)
	}
	later := first
	later.Epoch = f.Now.Add(time.Hour)
	later.AdvancedAt = later.Epoch
	later.Reason = domain.RevocationReasonOwnerDisabled
	if err := f.Epochs.Advance(ctx, later); err != nil {
		t.Fatalf("forward Advance: %v", err)
	}
	advanced, err := f.Epochs.FindByAgent(ctx, f.TenantA, f.AgentID)
	if err != nil || advanced == nil || !advanced.Epoch.Equal(later.Epoch) || advanced.Reason != later.Reason {
		t.Fatalf("FindByAgent after forward Advance = (%+v, %v)", advanced, err)
	}
}
