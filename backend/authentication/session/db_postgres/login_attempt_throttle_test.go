package db_postgres

import (
	"context"
	"testing"
	"time"

	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	pgfixtures "github.com/ambi/idmagic/backend/shared/storage/fixtures_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
)

// TestLoginAttemptThrottle は共有 login throttle (ADR-077 / ADR-139) を検証する。
// fixed-window でしきい値到達時にロックし、lockout 経過で解放、成功でクリア、tenant 分離、
// window リセットを memory adapter とのパリティで確認する。fail-closed は到達不能時の error 伝播で担保。
func TestLoginAttemptThrottle(t *testing.T) {
	db := pgtest.Require(t)
	tenant := pgfixtures.SeedTenant(t, db)
	other := pgfixtures.SeedTenant(t, db)
	configs := sessionports.LoginThrottleConfigs{
		Account: sessionports.LoginThrottleConfig{WindowSeconds: 120, MaxFailures: 3, LockoutSeconds: 300},
		IP:      sessionports.LoginThrottleConfig{WindowSeconds: 120, MaxFailures: 5, LockoutSeconds: 300},
	}
	throttle := &LoginAttemptThrottle{Pool: db, Configs: configs}
	ctx := withTenant(context.Background(), tenant)
	otherCtx := withTenant(context.Background(), other)
	now := pgtest.Now()
	acct := sessionports.LoginThrottleAccount

	if r, err := throttle.TryAcquire(ctx, acct, "alice", now); err != nil || !r.Allowed {
		t.Fatalf("initial acquire=%+v err=%v (want allowed)", r, err)
	}
	// two failures below threshold stay allowed.
	for i := range 2 {
		if r, err := throttle.RecordFailure(ctx, acct, "alice", now); err != nil || !r.Allowed || r.Locked {
			t.Fatalf("failure %d=%+v err=%v (want allowed)", i, r, err)
		}
	}
	// third failure hits MaxFailures=3 → locked.
	r, err := throttle.RecordFailure(ctx, acct, "alice", now)
	if err != nil || r.Allowed || !r.Locked || r.RetryAfterSeconds != 300 {
		t.Fatalf("threshold failure=%+v err=%v (want locked, retry=300)", r, err)
	}
	// TryAcquire while locked reports locked with a positive retry.
	if r, err := throttle.TryAcquire(ctx, acct, "alice", now); err != nil || r.Allowed || !r.Locked || r.RetryAfterSeconds <= 0 {
		t.Fatalf("locked acquire=%+v err=%v (want locked)", r, err)
	}
	// after lockout expiry, allowed again.
	if r, err := throttle.TryAcquire(ctx, acct, "alice", now.Add(301*time.Second)); err != nil || !r.Allowed {
		t.Fatalf("post-lockout acquire=%+v err=%v (want allowed)", r, err)
	}
	// other tenant is independent (isolation).
	if r, err := throttle.TryAcquire(otherCtx, acct, "alice", now); err != nil || !r.Allowed {
		t.Fatalf("cross-tenant acquire=%+v err=%v (want allowed)", r, err)
	}
	// RecordSuccess clears the counter/lock.
	if _, err := throttle.RecordFailure(ctx, acct, "bob", now); err != nil {
		t.Fatalf("bob failure: %v", err)
	}
	if err := throttle.RecordSuccess(ctx, acct, "bob"); err != nil {
		t.Fatalf("record success: %v", err)
	}
	if r, err := throttle.TryAcquire(ctx, acct, "bob", now); err != nil || !r.Allowed {
		t.Fatalf("post-success acquire=%+v err=%v (want allowed)", r, err)
	}
	// fixed-window: a failure after the window resets the counter (no lock at 2 failures).
	if _, err := throttle.RecordFailure(ctx, acct, "carol", now); err != nil {
		t.Fatalf("carol failure: %v", err)
	}
	if r, err := throttle.RecordFailure(ctx, acct, "carol", now.Add(121*time.Second)); err != nil || !r.Allowed || r.Locked {
		t.Fatalf("post-window failure=%+v err=%v (want allowed, counter reset)", r, err)
	}
	// GC removes fully-expired rows.
	if n, err := throttle.DeleteExpiredBatch(ctx, now.Add(3600*time.Second), 100); err != nil || n < 1 {
		t.Fatalf("gc n=%d err=%v (want >=1)", n, err)
	}
}
