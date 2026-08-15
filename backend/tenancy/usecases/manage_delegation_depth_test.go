package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func newDelegationDepthTenant(t *testing.T) (*memory.TenantRepository, string) {
	t.Helper()
	repo := memory.NewTenantRepository()
	created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return repo, created.ID
}

// REQ-TENANCY-021: 上書きは厳しい方向にのみ働く。
func TestUpdateMaxDelegationDepthOnlyTightens(t *testing.T) {
	floor := PolicyFloor{MinLength: 12, MaxLength: 128, HistoryDepth: 5}

	t.Run("a value below the system default is stored", func(t *testing.T) {
		repo, id := newDelegationDepthTenant(t)
		updated, err := Update(context.Background(), repo, id,
			UpdateInput{MaxDelegationDepth: new(1)}, floor, time.Now().UTC())
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.MaxDelegationDepth == nil || *updated.MaxDelegationDepth != 1 {
			t.Fatalf("MaxDelegationDepth = %v, want 1", updated.MaxDelegationDepth)
		}
		if updated.EffectiveMaxDelegationDepth() != 1 {
			t.Fatalf("EffectiveMaxDelegationDepth() = %d, want 1", updated.EffectiveMaxDelegationDepth())
		}
		// 上書きは永続化され、読み直しても同じ値が返る。
		reloaded, err := repo.FindByID(context.Background(), id)
		if err != nil || reloaded.MaxDelegationDepth == nil || *reloaded.MaxDelegationDepth != 1 {
			t.Fatalf("reloaded = %v (err %v), want the stored override", reloaded.MaxDelegationDepth, err)
		}
	})

	t.Run("a value above the system default is rejected", func(t *testing.T) {
		repo, id := newDelegationDepthTenant(t)
		_, err := Update(context.Background(), repo, id,
			UpdateInput{MaxDelegationDepth: new(domain.DefaultMaxDelegationDepth + 1)}, floor, time.Now().UTC())
		if !errors.Is(err, ErrPolicyOverrideWeaker) {
			t.Fatalf("err = %v, want %v", err, ErrPolicyOverrideWeaker)
		}
		stored, err := repo.FindByID(context.Background(), id)
		if err != nil || stored.MaxDelegationDepth != nil {
			t.Fatalf("a rejected override must not be persisted: %v (err %v)", stored.MaxDelegationDepth, err)
		}
	})

	t.Run("a value below one is rejected", func(t *testing.T) {
		repo, id := newDelegationDepthTenant(t)
		if _, err := Update(context.Background(), repo, id,
			UpdateInput{MaxDelegationDepth: new(-1)}, floor, time.Now().UTC()); !errors.Is(err, ErrPolicyOverrideWeaker) {
			t.Fatalf("err = %v, want %v", err, ErrPolicyOverrideWeaker)
		}
	})

	t.Run("zero clears the override back to the system default", func(t *testing.T) {
		repo, id := newDelegationDepthTenant(t)
		if _, err := Update(context.Background(), repo, id,
			UpdateInput{MaxDelegationDepth: new(2)}, floor, time.Now().UTC()); err != nil {
			t.Fatalf("Update: %v", err)
		}
		updated, err := Update(context.Background(), repo, id,
			UpdateInput{MaxDelegationDepth: new(0)}, floor, time.Now().UTC())
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.MaxDelegationDepth != nil {
			t.Fatalf("MaxDelegationDepth = %v, want nil after clearing", updated.MaxDelegationDepth)
		}
		if updated.EffectiveMaxDelegationDepth() != domain.DefaultMaxDelegationDepth {
			t.Fatalf("EffectiveMaxDelegationDepth() = %d, want the system default",
				updated.EffectiveMaxDelegationDepth())
		}
	})

	t.Run("an omitted field leaves the existing override alone", func(t *testing.T) {
		repo, id := newDelegationDepthTenant(t)
		if _, err := Update(context.Background(), repo, id,
			UpdateInput{MaxDelegationDepth: new(2)}, floor, time.Now().UTC()); err != nil {
			t.Fatalf("Update: %v", err)
		}
		name := "Acme Inc."
		updated, err := Update(context.Background(), repo, id,
			UpdateInput{DisplayName: &name}, floor, time.Now().UTC())
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if updated.MaxDelegationDepth == nil || *updated.MaxDelegationDepth != 2 {
			t.Fatalf("MaxDelegationDepth = %v, want the previous override preserved", updated.MaxDelegationDepth)
		}
	})
}

// 既定値を変えると既存の委譲チェーンが壊れる。上書きの無いテナントは現行と同じ 3 を継承する。
func TestTenantWithoutOverrideKeepsTheProductDefault(t *testing.T) {
	repo, id := newDelegationDepthTenant(t)
	tenant, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if tenant.MaxDelegationDepth != nil {
		t.Fatalf("a new tenant must carry no override, got %v", tenant.MaxDelegationDepth)
	}
	if got := tenant.EffectiveMaxDelegationDepth(); got != 3 {
		t.Fatalf("EffectiveMaxDelegationDepth() = %d, want 3 (unchanged from before the policy existed)", got)
	}
}
