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
