package db_postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"time"

	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
)

// RateLimiter counts requests against the shared endpoint_rate_limit_counters row in
// PostgreSQL (ADR-157). Unlike the login throttle, every call to Allow consumes budget
// regardless of the request's outcome, so the read-modify-write is simpler: no lockout state,
// just a fixed-window counter compared against the configured threshold on every call.
// Reaching an unreachable store returns an error, letting the caller fail closed.
type RateLimiter struct {
	Pool    sharedpg.DB
	Configs rlports.RateLimitConfigs
}

func hashRateLimitKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (l *RateLimiter) Allow(ctx context.Context, policyID, key string, now time.Time) (rlports.RateLimitResult, error) {
	config, ok := l.Configs[policyID]
	if !ok {
		return rlports.RateLimitResult{}, errors.New("ratelimit: unknown policy " + policyID)
	}
	tenantID := tenancy.TenantID(ctx)
	hash := hashRateLimitKey(key)

	tx, err := l.Pool.Begin(ctx)
	if err != nil {
		return rlports.RateLimitResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)

	row, err := q.LockRateLimitCounter(ctx, LockRateLimitCounterParams{TenantID: tenantID, PolicyID: policyID, KeyHash: hash})
	var (
		count     int
		windowExp time.Time
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		count = 1
		windowExp = now.Add(time.Duration(config.WindowSeconds) * time.Second)
	case err != nil:
		return rlports.RateLimitResult{}, err
	case !now.Before(row.WindowExpiresAt):
		// window elapsed: reset the counter.
		count = 1
		windowExp = now.Add(time.Duration(config.WindowSeconds) * time.Second)
	default:
		count = int(row.Count) + 1
		windowExp = row.WindowExpiresAt
	}

	if err := q.UpsertRateLimitCounter(ctx, UpsertRateLimitCounterParams{
		TenantID:        tenantID,
		PolicyID:        policyID,
		KeyHash:         hash,
		Count:           int32(count),
		WindowExpiresAt: windowExp,
	}); err != nil {
		return rlports.RateLimitResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return rlports.RateLimitResult{}, err
	}
	if count > config.MaxRequests {
		return rlports.RateLimitResult{
			Allowed:           false,
			RetryAfterSeconds: int(math.Ceil(windowExp.Sub(now).Seconds())),
		}, nil
	}
	return rlports.RateLimitResult{Allowed: true}, nil
}

func (l *RateLimiter) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(l.Pool).DeleteExpiredRateLimitCountersBatch(ctx, DeleteExpiredRateLimitCountersBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
