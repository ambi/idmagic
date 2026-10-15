package testing_contract

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

// QuotaFixture は、使用量を持たないテナント一つと、その上限を扱うリポジトリである。
type QuotaFixture struct {
	Repository ports.QuotaRepository
	TenantID   string
}

type NewQuotaFixture func(t *testing.T) QuotaFixture

// RunQuota は、Hard Quota の実効値と上限の更新について、どのバックエンドも同じ結果を返すことを
// 確かめる。ケースごとに新しいテナントを使い、前のケースの使用量を持ち込まない。
func RunQuota(t *testing.T, newFixture NewQuotaFixture) {
	t.Helper()
	ctx := context.Background()

	t.Run("override limits the resource and a refused increment keeps the usage", func(t *testing.T) {
		f := newFixture(t)
		one := 1
		if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{Users: &one}); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceUsers, 1); err != nil {
			t.Fatalf("first increment: %v", err)
		}
		requireQuotaExceeded(t, f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceUsers, 1))
		requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.Users }, 1)
	})

	t.Run("without an override the default limit applies", func(t *testing.T) {
		f := newFixture(t)
		limit := domain.DefaultTenantQuota[domain.ResourceActiveJobs]
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceActiveJobs, limit); err != nil {
			t.Fatalf("increment up to the default %d: %v", limit, err)
		}
		requireQuotaExceeded(t, f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceActiveJobs, 1))
		requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.ActiveJobs }, limit)
	})

	t.Run("an unknown resource fails and counts nothing", func(t *testing.T) {
		f := newFixture(t)
		err := f.Repository.CheckAndIncrement(ctx, f.TenantID, "widgets", 1)
		if err == nil {
			t.Fatal("increment of an unknown resource succeeded")
		}
		if _, exceeded := errors.AsType[*domain.QuotaExceededError](err); exceeded {
			t.Fatalf("unknown resource reported as quota exceeded: %v", err)
		}
		usage, err := f.Repository.GetUsage(ctx, f.TenantID)
		if err != nil {
			t.Fatalf("GetUsage: %v", err)
		}
		if *usage != (domain.TenantUsage{}) {
			t.Fatalf("usage = %+v, want zero", usage)
		}
	})

	t.Run("a decrement does not go below zero", func(t *testing.T) {
		f := newFixture(t)
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceGroups, 1); err != nil {
			t.Fatalf("increment: %v", err)
		}
		if err := f.Repository.Decrement(ctx, f.TenantID, domain.ResourceGroups, 5); err != nil {
			t.Fatalf("Decrement: %v", err)
		}
		requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.Groups }, 0)
	})

	t.Run("an update replaces every override", func(t *testing.T) {
		f := newFixture(t)
		users, groups := 20000, 5
		if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{Users: &users, Groups: &groups}); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
		users = 30000
		if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{Users: &users}); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
		quota, err := f.Repository.GetQuota(ctx, f.TenantID)
		if err != nil {
			t.Fatalf("GetQuota: %v", err)
		}
		if quota.Users == nil || *quota.Users != 30000 || quota.Groups != nil {
			t.Fatalf("quota = users %v, groups %v; want 30000 and no groups override", quota.Users, quota.Groups)
		}
		if got := quota.EffectiveLimit(domain.ResourceGroups); got != domain.DefaultTenantQuota[domain.ResourceGroups] {
			t.Fatalf("effective groups limit = %d, want the default", got)
		}
	})

	t.Run("a limit below the usage is kept and refuses the next creation", func(t *testing.T) {
		f := newFixture(t)
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceGroups, 3); err != nil {
			t.Fatalf("increment: %v", err)
		}
		two := 2
		if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{Groups: &two}); err != nil {
			t.Fatalf("SetQuota below usage: %v", err)
		}
		requireQuotaExceeded(t, f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceGroups, 1))
		requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.Groups }, 3)
	})
}

func requireQuotaExceeded(t *testing.T, err error) {
	t.Helper()
	if _, ok := errors.AsType[*domain.QuotaExceededError](err); !ok {
		t.Fatalf("err = %v, want QuotaExceededError", err)
	}
}

func requireUsage(ctx context.Context, t *testing.T, f QuotaFixture, read func(*domain.TenantUsage) int, want int) {
	t.Helper()
	usage, err := f.Repository.GetUsage(ctx, f.TenantID)
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if got := read(usage); got != want {
		t.Fatalf("usage = %d, want %d", got, want)
	}
}
