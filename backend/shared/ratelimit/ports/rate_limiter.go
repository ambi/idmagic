package ports

import (
	"context"
	"time"
)

type RateLimitResult struct {
	Allowed           bool
	RetryAfterSeconds int
}

// RateLimitPolicyConfig is a fixed-window (max_requests, window_seconds) threshold for one route.
type RateLimitPolicyConfig struct {
	MaxRequests   int
	WindowSeconds int
}

// RateLimitConfigs maps a policy id (e.g. "token", "authorize", "par", "device_authorization",
// "password_reset", "login") to its threshold. memory / postgres adapters consume the same
// port type (ADR-157).
type RateLimitConfigs map[string]RateLimitPolicyConfig

type RateLimiter interface {
	// Allow atomically increments the fixed-window counter for (tenant, policyID, key) and
	// reports whether this request is within the configured threshold. Unlike the login
	// throttle, every call consumes budget regardless of the request's outcome. Tenant scoping
	// (postgres adapter) is resolved from ctx via tenancy.TenantID, matching LoginAttemptThrottle.
	Allow(ctx context.Context, policyID, key string, now time.Time) (RateLimitResult, error)
}

// Module bundles the shared rate limiter capability for composition roots. It is a
// cross-cutting technical capability (ADR-157), not owned by any single bounded context, so it
// sits alongside the other shared/* modules (e.g. notification) rather than nested inside a
// context Module.
type Module struct {
	NewRateLimiter func(configs RateLimitConfigs) RateLimiter
	RateLimiter    RateLimiter
}
