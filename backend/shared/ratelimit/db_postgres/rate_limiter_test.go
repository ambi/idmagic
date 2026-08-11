package db_postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func withTenant(ctx context.Context, tenant *tenancydomain.Tenant) context.Context {
	return tenancy.WithTenant(ctx, tenant, "", "")
}

// TestRateLimiter verifies the shared endpoint rate limiter: fixed-window Allow
// blocks at threshold, resets after window expiry, isolates by tenant/policy/key, and GC
// reclaims expired rows. Parity with the memory adapter's contract.
func TestRateLimiter(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	limiter := &RateLimiter{Pool: db, Configs: rlports.RateLimitConfigs{
		"token": {MaxRequests: 3, WindowSeconds: 120},
	}}
	ctx := withTenant(context.Background(), tenant)
	otherCtx := withTenant(context.Background(), other)
	now := pgtest.Now()

	for i := range 3 {
		if r, err := limiter.Allow(ctx, "token", "client-1|203.0.113.1", now); err != nil || !r.Allowed {
			t.Fatalf("request %d=%+v err=%v (want allowed)", i+1, r, err)
		}
	}
	r, err := limiter.Allow(ctx, "token", "client-1|203.0.113.1", now)
	if err != nil || r.Allowed || r.RetryAfterSeconds <= 0 {
		t.Fatalf("4th request=%+v err=%v (want blocked with positive retry)", r, err)
	}
	// other tenant is independent (isolation).
	if r, err := limiter.Allow(otherCtx, "token", "client-1|203.0.113.1", now); err != nil || !r.Allowed {
		t.Fatalf("cross-tenant request=%+v err=%v (want allowed)", r, err)
	}
	// after window expiry, allowed again (counter reset).
	if r, err := limiter.Allow(ctx, "token", "client-1|203.0.113.1", now.Add(121*time.Second)); err != nil || !r.Allowed {
		t.Fatalf("post-window request=%+v err=%v (want allowed)", r, err)
	}
	// GC removes fully-expired rows.
	if n, err := limiter.DeleteExpiredBatch(ctx, now.Add(3600*time.Second), 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}

// TestRateLimiterConcurrentAllow proves the tx + SELECT FOR UPDATE read-modify-write is
// atomic under real concurrency: with MaxRequests=8 and 16 concurrent callers on the same
// key, exactly 8 must be allowed, matching wi-278's 16-way concurrency proof for the same
// CAS pattern (login_throttle_counters).
func TestRateLimiterConcurrentAllow(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	limiter := &RateLimiter{Pool: db, Configs: rlports.RateLimitConfigs{
		"token": {MaxRequests: 8, WindowSeconds: 120},
	}}
	ctx := withTenant(context.Background(), tenant)
	now := pgtest.Now()

	const concurrency = 16
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			r, err := limiter.Allow(ctx, "token", "shared-key", now)
			if err != nil {
				t.Errorf("Allow: %v", err)
				return
			}
			if r.Allowed {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != 8 {
		t.Fatalf("allowed=%d, want exactly 8 (MaxRequests) out of %d concurrent callers", allowed, concurrency)
	}
}
