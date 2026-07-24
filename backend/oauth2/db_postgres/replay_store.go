package db_postgres

import (
	"context"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
)

// ReplayStore は DPoP / private_key_jwt assertion の jti リプレイ予約を PostgreSQL に持つ
// (ADR-139)。kind 列で 1 テーブル oauth2_replay_jtis を dpop / client_assertion に
// 名前空間分けする。同一構造体が DpopReplayStore /
// ClientAssertionReplayStore 両 port を構造的に満たす。tenant は ctx から解決する。
// SETNX + TTL を INSERT ... ON CONFLICT DO UPDATE ... WHERE expires_at <= now RETURNING で写す:
// live な予約は 0 行 (= false)、期限切れ / 未存在は 1 行 (= true)。
type ReplayStore struct {
	Pool sharedpg.DB
	Kind string
}

func (s *ReplayStore) RecordIfNew(ctx context.Context, jti string, windowSeconds int, now time.Time) (bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := New(s.Pool).ReserveOauth2ReplayJTI(ctx, ReserveOauth2ReplayJTIParams{
		TenantID:     tenancy.TenantID(ctx),
		Kind:         s.Kind,
		Jti:          jti,
		NewExpiresAt: now.Add(time.Duration(windowSeconds) * time.Second),
		Now:          now,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *ReplayStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredOauth2ReplayJTIsBatch(ctx, DeleteExpiredOauth2ReplayJTIsBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: small housekeeping batch size
	})
	return int(deleted), err
}
