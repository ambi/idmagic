package db_memory

import (
	"context"
	"testing"
	"time"

	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
)

func TestRateLimiterAllowsWithinThresholdThenBlocks(t *testing.T) {
	limiter := NewRateLimiter(rlports.RateLimitConfigs{
		"token": {MaxRequests: 3, WindowSeconds: 60},
	})
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := range 3 {
		result, err := limiter.Allow(ctx, "token", "client-1|203.0.113.1", now)
		if err != nil || !result.Allowed {
			t.Fatalf("request %d: result=%+v err=%v", i+1, result, err)
		}
	}
	result, err := limiter.Allow(ctx, "token", "client-1|203.0.113.1", now)
	if err != nil || result.Allowed || result.RetryAfterSeconds <= 0 {
		t.Fatalf("4th request should be blocked: result=%+v err=%v", result, err)
	}
}

func TestRateLimiterWindowExpiryResetsCounter(t *testing.T) {
	limiter := NewRateLimiter(rlports.RateLimitConfigs{
		"token": {MaxRequests: 1, WindowSeconds: 60},
	})
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if result, err := limiter.Allow(ctx, "token", "key", now); err != nil || !result.Allowed {
		t.Fatalf("first request: result=%+v err=%v", result, err)
	}
	if result, err := limiter.Allow(ctx, "token", "key", now); err != nil || result.Allowed {
		t.Fatalf("second request within window should be blocked: result=%+v err=%v", result, err)
	}
	if result, err := limiter.Allow(ctx, "token", "key", now.Add(61*time.Second)); err != nil || !result.Allowed {
		t.Fatalf("request after window expiry should be allowed: result=%+v err=%v", result, err)
	}
}

func TestRateLimiterIsolatesByPolicyAndKey(t *testing.T) {
	limiter := NewRateLimiter(rlports.RateLimitConfigs{
		"token":     {MaxRequests: 1, WindowSeconds: 60},
		"authorize": {MaxRequests: 1, WindowSeconds: 60},
	})
	ctx := context.Background()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if result, err := limiter.Allow(ctx, "token", "key-a", now); err != nil || !result.Allowed {
		t.Fatalf("key-a first request: result=%+v err=%v", result, err)
	}
	if result, err := limiter.Allow(ctx, "token", "key-b", now); err != nil || !result.Allowed {
		t.Fatalf("different key should not share the counter: result=%+v err=%v", result, err)
	}
	if result, err := limiter.Allow(ctx, "authorize", "key-a", now); err != nil || !result.Allowed {
		t.Fatalf("different policy should not share the counter: result=%+v err=%v", result, err)
	}
}

func TestRateLimiterUnknownPolicyFailsClosed(t *testing.T) {
	limiter := NewRateLimiter(rlports.RateLimitConfigs{})
	ctx := context.Background()
	result, err := limiter.Allow(ctx, "unknown-policy", "key", time.Now().UTC())
	if err == nil {
		t.Fatalf("expected error for unknown policy, got result=%+v", result)
	}
}
