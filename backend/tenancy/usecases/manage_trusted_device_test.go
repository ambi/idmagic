package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"
)

func newTrustedDeviceTenant(t *testing.T) (*memory.TenantRepository, string) {
	t.Helper()
	repo := memory.NewTenantRepository()
	created, err := Create(context.Background(), repo, "acme", "Acme", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return repo, created.ID
}

// 信頼済みデバイスは既定で無効であり、テナントが正の値を明示したときだけ有効になる (wi-91)。
func TestTrustedDeviceMaxAgeDefaultsToDisabled(t *testing.T) {
	repo, id := newTrustedDeviceTenant(t)
	tenant, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if tenant.TrustedDeviceMaxAgeSeconds != nil {
		t.Fatalf("TrustedDeviceMaxAgeSeconds = %v, want unset", tenant.TrustedDeviceMaxAgeSeconds)
	}
	if got := tenant.EffectiveTrustedDeviceMaxAge(); got != 0 {
		t.Fatalf("EffectiveTrustedDeviceMaxAge() = %v, want 0 (disabled)", got)
	}
}

// 上限以内の値は保存でき、0 は機能無効へ戻す。上限超過と負値は拒否する (wi-91)。
func TestUpdateTrustedDeviceMaxAgeAcceptsRangeAndRejectsOverflow(t *testing.T) {
	repo, id := newTrustedDeviceTenant(t)
	ctx, floor, now := context.Background(), PolicyFloor{}, time.Now().UTC()
	week := 7 * 24 * 60 * 60

	updated, err := Update(ctx, repo, id, UpdateInput{TrustedDeviceMaxAgeSeconds: &week}, floor, now)
	if err != nil {
		t.Fatal(err)
	}
	if updated.EffectiveTrustedDeviceMaxAge() != 7*24*time.Hour {
		t.Fatalf("EffectiveTrustedDeviceMaxAge() = %v, want 7 days", updated.EffectiveTrustedDeviceMaxAge())
	}

	over := domain.TrustedDeviceMaxAgeCeilingSeconds + 1
	if _, err := Update(
		ctx, repo, id, UpdateInput{TrustedDeviceMaxAgeSeconds: &over}, floor, now,
	); !errors.Is(err, ErrPolicyOverrideWeaker) {
		t.Fatalf("err = %v, want ErrPolicyOverrideWeaker above the ceiling", err)
	}
	stored, err := repo.FindByID(ctx, id)
	if err != nil || stored.EffectiveTrustedDeviceMaxAge() != 7*24*time.Hour {
		t.Fatalf("a rejected value must not be persisted: %v (err %v)", stored.TrustedDeviceMaxAgeSeconds, err)
	}

	off := 0
	cleared, err := Update(ctx, repo, id, UpdateInput{TrustedDeviceMaxAgeSeconds: &off}, floor, now)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.TrustedDeviceMaxAgeSeconds != nil || cleared.EffectiveTrustedDeviceMaxAge() != 0 {
		t.Fatalf("0 must disable the feature, got %v", cleared.TrustedDeviceMaxAgeSeconds)
	}
}

// 壊れた値が保存されていても、読み出し側は 0 (無効) として読み MFA を弱めない (wi-91)。
func TestEffectiveTrustedDeviceMaxAgeFailsClosedOnBrokenValues(t *testing.T) {
	for _, seconds := range []int{-1, domain.TrustedDeviceMaxAgeCeilingSeconds + 1} {
		tenant := domain.Tenant{TrustedDeviceMaxAgeSeconds: &seconds}
		if got := tenant.EffectiveTrustedDeviceMaxAge(); got != 0 {
			t.Fatalf("EffectiveTrustedDeviceMaxAge(%d) = %v, want 0", seconds, got)
		}
	}
}
