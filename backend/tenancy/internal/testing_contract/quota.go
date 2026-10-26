package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

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

	runRecountUsage(t, newFixture)
}

// runRecountUsage は、User の使用量の再集計が数えた件数で使用量を置き換え、数えている間の
// 加算を失わないことを確かめる (REQ-TENANCY-049)。
func runRecountUsage(t *testing.T, newFixture NewQuotaFixture) {
	t.Helper()
	ctx := context.Background()
	users := func(u *domain.TenantUsage) int { return u.Users }
	counted := func(n int) func(context.Context) (int, error) {
		return func(context.Context) (int, error) { return n, nil }
	}

	t.Run("recount replaces the users usage and keeps the other resources", func(t *testing.T) {
		f := newFixture(t)
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceUsers, 3); err != nil {
			t.Fatalf("increment users: %v", err)
		}
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceGroups, 2); err != nil {
			t.Fatalf("increment groups: %v", err)
		}
		previous, err := f.Repository.RecountUsage(ctx, f.TenantID, domain.ResourceUsers, counted(1))
		if err != nil || previous != 3 {
			t.Fatalf("RecountUsage = %d, %v; want the previous usage 3", previous, err)
		}
		requireUsage(ctx, t, f, users, 1)
		requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.Groups }, 2)
	})

	t.Run("recount of a tenant without usage stores the count", func(t *testing.T) {
		f := newFixture(t)
		previous, err := f.Repository.RecountUsage(ctx, f.TenantID, domain.ResourceUsers, counted(2))
		if err != nil || previous != 0 {
			t.Fatalf("RecountUsage = %d, %v; want the previous usage 0", previous, err)
		}
		requireUsage(ctx, t, f, users, 2)
	})

	t.Run("a count above the limit is stored and refuses the next creation", func(t *testing.T) {
		f := newFixture(t)
		one := 1
		if err := f.Repository.SetQuota(ctx, f.TenantID, &domain.TenantQuota{Users: &one}); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
		if _, err := f.Repository.RecountUsage(ctx, f.TenantID, domain.ResourceUsers, counted(2)); err != nil {
			t.Fatalf("RecountUsage: %v", err)
		}
		requireUsage(ctx, t, f, users, 2)
		requireQuotaExceeded(t, f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceUsers, 1))
	})

	t.Run("a failed or negative count keeps the usage", func(t *testing.T) {
		f := newFixture(t)
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceUsers, 3); err != nil {
			t.Fatalf("increment: %v", err)
		}
		failing := func(context.Context) (int, error) { return 0, errors.New("count unavailable") }
		for name, count := range map[string]func(context.Context) (int, error){"failed": failing, "negative": counted(-1)} {
			if _, err := f.Repository.RecountUsage(ctx, f.TenantID, domain.ResourceUsers, count); err == nil {
				t.Fatalf("%s count was accepted", name)
			}
			requireUsage(ctx, t, f, users, 3)
		}
	})

	t.Run("recount of a resource other than users fails and changes nothing", func(t *testing.T) {
		f := newFixture(t)
		if err := f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceGroups, 2); err != nil {
			t.Fatalf("increment: %v", err)
		}
		if _, err := f.Repository.RecountUsage(ctx, f.TenantID, domain.ResourceGroups, counted(0)); err == nil {
			t.Fatal("recount of groups succeeded")
		}
		requireUsage(ctx, t, f, func(u *domain.TenantUsage) int { return u.Groups }, 2)
	})

	// 数えている間の加算が置き換えの前に確定すれば、置き換えがその加算を消す。加算が数え終わる
	// まで待たされ、置き換えた件数に加わることを確かめる。
	t.Run("an increment during the count is applied after the replacement", func(t *testing.T) {
		f := newFixture(t)
		incremented := make(chan error, 1)
		count := func(context.Context) (int, error) {
			go func() { incremented <- f.Repository.CheckAndIncrement(ctx, f.TenantID, domain.ResourceUsers, 1) }()
			select {
			case err := <-incremented:
				incremented <- err
			case <-time.After(200 * time.Millisecond):
			}
			return 5, nil
		}
		if _, err := f.Repository.RecountUsage(ctx, f.TenantID, domain.ResourceUsers, count); err != nil {
			t.Fatalf("RecountUsage: %v", err)
		}
		if err := <-incremented; err != nil {
			t.Fatalf("concurrent increment: %v", err)
		}
		requireUsage(ctx, t, f, users, 6)
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
