package ports

import (
	"context"
	"time"
)

type LoginThrottleKind string

const (
	LoginThrottleAccount LoginThrottleKind = "account"
	LoginThrottleIP      LoginThrottleKind = "ip"
)

type LoginThrottleResult struct {
	Allowed           bool
	Locked            bool
	RetryAfterSeconds int
}

// LoginThrottleConfig は 1 軸 (per-account または per-IP) の fixed-window しきい値。
type LoginThrottleConfig struct {
	MaxFailures    int
	WindowSeconds  int
	LockoutSeconds int
}

// LoginThrottleConfigs は per-account / per-IP の両軸のしきい値を束ねる。
// memory / valkey いずれの adapter も同一の port 型を消費する (ADR-077)。
type LoginThrottleConfigs struct {
	Account LoginThrottleConfig
	IP      LoginThrottleConfig
}

type LoginAttemptThrottle interface {
	TryAcquire(ctx context.Context, kind LoginThrottleKind, key string, now time.Time) (LoginThrottleResult, error)
	RecordFailure(ctx context.Context, kind LoginThrottleKind, key string, now time.Time) (LoginThrottleResult, error)
	RecordSuccess(ctx context.Context, kind LoginThrottleKind, key string) error
}
