// Package testing_contract defines the shared rate-limiter contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/ratelimit/ports"
)

type Fixture struct {
	Limiter ports.RateLimiter
	Context context.Context
	Now     time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	for range 2 {
		result, err := f.Limiter.Allow(f.Context, "contract", "key", f.Now)
		if err != nil || !result.Allowed {
			t.Fatalf("Allow before limit = (%+v, %v)", result, err)
		}
	}
	result, err := f.Limiter.Allow(f.Context, "contract", "key", f.Now)
	if err != nil || result.Allowed || result.RetryAfterSeconds <= 0 {
		t.Fatalf("Allow at limit = (%+v, %v)", result, err)
	}
	if result, err := f.Limiter.Allow(f.Context, "other-policy", "key", f.Now); err != nil || !result.Allowed {
		t.Fatalf("policy isolation = (%+v, %v)", result, err)
	}
	if result, err := f.Limiter.Allow(f.Context, "contract", "key", f.Now.Add(61*time.Second)); err != nil || !result.Allowed {
		t.Fatalf("window reset = (%+v, %v)", result, err)
	}
}

// RunAllowWithin checks that the per-request threshold reaches the store rather than a fixed config.
// The agent issuance limit of REQ-OAUTH2-074 relies on this.
func RunAllowWithin(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	config := ports.RateLimitPolicyConfig{MaxRequests: 1, WindowSeconds: 30}
	if result, err := f.Limiter.AllowWithin(f.Context, "unconfigured", "key", config, f.Now); err != nil || !result.Allowed {
		t.Fatalf("AllowWithin before limit = (%+v, %v)", result, err)
	}
	result, err := f.Limiter.AllowWithin(f.Context, "unconfigured", "key", config, f.Now)
	if err != nil || result.Allowed || result.RetryAfterSeconds <= 0 {
		t.Fatalf("AllowWithin at limit = (%+v, %v)", result, err)
	}
	if result, err := f.Limiter.AllowWithin(f.Context, "unconfigured", "key", config, f.Now.Add(31*time.Second)); err != nil || !result.Allowed {
		t.Fatalf("AllowWithin window reset = (%+v, %v)", result, err)
	}
}
