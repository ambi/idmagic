package db_postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"time"

	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LoginAttemptThrottle は login throttle の counter / lock を PostgreSQL の共有行で数える。
// ADR-077 の共有ストア化・fail-closed・SHA-256 識別子・tenant scoping を維持しつつ、機構だけを
// Valkey から Postgres へ移す (ADR-139: Postgres は既に hard dependency で依存を増やさない)。
// fixed-window の counter と lockout を 1 行 (failures / window_expires_at / locked_until) に統合し、
// RecordFailure は tx + SELECT FOR UPDATE の read-modify-write で原子化する (Valkey Lua の写し)。
// 到達不能時はエラーを返し、呼び出し側で fail-closed に倒れる。
type LoginAttemptThrottle struct {
	Pool    sharedpg.DB
	Configs sessionports.LoginThrottleConfigs
}

func (t *LoginAttemptThrottle) config(kind sessionports.LoginThrottleKind) sessionports.LoginThrottleConfig {
	if kind == sessionports.LoginThrottleIP {
		return t.Configs.IP
	}
	return t.Configs.Account
}

func hashThrottleIdentifier(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (t *LoginAttemptThrottle) TryAcquire(ctx context.Context, kind sessionports.LoginThrottleKind, key string, now time.Time) (sessionports.LoginThrottleResult, error) {
	lockedUntil, err := New(t.Pool).GetThrottleLock(ctx, GetThrottleLockParams{
		TenantID:       tenancy.TenantID(ctx),
		Kind:           string(kind),
		IdentifierHash: hashThrottleIdentifier(key),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionports.LoginThrottleResult{Allowed: true}, nil
	}
	if err != nil {
		return sessionports.LoginThrottleResult{}, err
	}
	if !lockedUntil.Valid {
		return sessionports.LoginThrottleResult{Allowed: true}, nil
	}
	remaining := lockedUntil.Time.Sub(now)
	if remaining <= 0 {
		return sessionports.LoginThrottleResult{Allowed: true}, nil
	}
	return sessionports.LoginThrottleResult{
		Allowed: false, Locked: true,
		RetryAfterSeconds: int(math.Ceil(remaining.Seconds())),
	}, nil
}

func (t *LoginAttemptThrottle) RecordFailure(ctx context.Context, kind sessionports.LoginThrottleKind, key string, now time.Time) (sessionports.LoginThrottleResult, error) {
	config := t.config(kind)
	tenantID := tenancy.TenantID(ctx)
	hash := hashThrottleIdentifier(key)

	tx, err := t.Pool.Begin(ctx)
	if err != nil {
		return sessionports.LoginThrottleResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)

	row, err := q.LockThrottleCounter(ctx, LockThrottleCounterParams{TenantID: tenantID, Kind: string(kind), IdentifierHash: hash})
	var (
		failures    int
		windowExp   time.Time
		lockedUntil pgtype.Timestamptz
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		failures = 1
		windowExp = now.Add(time.Duration(config.WindowSeconds) * time.Second)
	case err != nil:
		return sessionports.LoginThrottleResult{}, err
	case !now.Before(row.WindowExpiresAt):
		// window 経過: counter をリセットする。既存の lock は保持する。
		failures = 1
		windowExp = now.Add(time.Duration(config.WindowSeconds) * time.Second)
		lockedUntil = row.LockedUntil
	default:
		failures = int(row.Failures) + 1
		windowExp = row.WindowExpiresAt
		lockedUntil = row.LockedUntil
	}

	result := sessionports.LoginThrottleResult{Allowed: true}
	if failures >= config.MaxFailures {
		// しきい値到達: lockout を張り counter をリセットする (Valkey の DEL counter + SET lock)。
		lockedUntil = pgtype.Timestamptz{Time: now.Add(time.Duration(config.LockoutSeconds) * time.Second), Valid: true}
		failures = 0
		windowExp = now.Add(time.Duration(config.WindowSeconds) * time.Second)
		result = sessionports.LoginThrottleResult{Allowed: false, Locked: true, RetryAfterSeconds: config.LockoutSeconds}
	}

	if err := q.UpsertThrottleCounter(ctx, UpsertThrottleCounterParams{
		TenantID:        tenantID,
		Kind:            string(kind),
		IdentifierHash:  hash,
		Failures:        int32(failures),
		WindowExpiresAt: windowExp,
		LockedUntil:     lockedUntil,
	}); err != nil {
		return sessionports.LoginThrottleResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sessionports.LoginThrottleResult{}, err
	}
	return result, nil
}

func (t *LoginAttemptThrottle) RecordSuccess(ctx context.Context, kind sessionports.LoginThrottleKind, key string) error {
	return New(t.Pool).DeleteThrottleCounter(ctx, DeleteThrottleCounterParams{
		TenantID:       tenancy.TenantID(ctx),
		Kind:           string(kind),
		IdentifierHash: hashThrottleIdentifier(key),
	})
}

func (t *LoginAttemptThrottle) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(t.Pool).DeleteExpiredThrottleCountersBatch(ctx, DeleteExpiredThrottleCountersBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
