package db_postgres

import (
	"context"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// AuthnRequestReplayStore は SAML AuthnRequest リプレイ予約を PostgreSQL に持つ (ADR-139)。
// port の SETNX + TTL 意味論を INSERT ... ON CONFLICT DO UPDATE ... WHERE expires_at <= now で
// 写す: live な予約は WHERE が false になり 0 行 (= 予約失敗)、期限切れ / 未存在は 1 行
// (= 予約成功) を返す。到達不能時はエラーを返し、呼び出し側で fail-closed に倒れる
// (constraint SAML2Core-BearerAssertion: 同一 tenant / SP / request ID は一度だけ)。
type AuthnRequestReplayStore struct{ Pool sharedpg.DB }

func (s *AuthnRequestReplayStore) RecordIfNew(
	ctx context.Context,
	tenantID, entityID, requestID string,
	ttl time.Duration,
	now time.Time,
) (bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := New(s.Pool).ReserveSamlAuthnRequestReplay(ctx, ReserveSamlAuthnRequestReplayParams{
		TenantID:     tenantID,
		EntityID:     entityID,
		RequestID:    requestID,
		NewExpiresAt: now.Add(ttl),
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

func (s *AuthnRequestReplayStore) DeleteExpiredBatch(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	deleted, err := New(s.Pool).DeleteExpiredSamlAuthnRequestReplaysBatch(ctx, DeleteExpiredSamlAuthnRequestReplaysBatchParams{
		Cutoff:     cutoff,
		BatchLimit: int32(limit), //nolint:gosec // G115: limit is a small housekeeping batch size, well under int32 max
	})
	return int(deleted), err
}
