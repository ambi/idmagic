package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/idempotency"
	postgres "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

type Store struct{ Pool postgres.DB }

func (s *Store) Claim(ctx context.Context, candidate idempotency.Record, _ time.Time) (idempotency.Record, bool, error) {
	queries := New(s.Pool)
	// 競合した受付と、その後の期限切れ削除が重なった場合だけ取得し直す。
	for range 3 {
		row, err := queries.ClaimIdempotencyRecord(ctx, ClaimIdempotencyRecordParams{
			ID: candidate.ID, TenantID: candidate.TenantID, PrincipalID: candidate.PrincipalID,
			KeyHash: candidate.KeyHash, Fingerprint: candidate.Fingerprint, CreatedAt: candidate.CreatedAt, ExpiresAt: candidate.ExpiresAt,
		})
		if err == nil {
			return fromRow(row), true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return idempotency.Record{}, false, err
		}
		row, err = queries.FindIdempotencyRecord(ctx, FindIdempotencyRecordParams{
			TenantID: candidate.TenantID, PrincipalID: candidate.PrincipalID, KeyHash: candidate.KeyHash,
		})
		if err == nil {
			return fromRow(row), false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return idempotency.Record{}, false, err
		}
	}
	return idempotency.Record{}, false, errors.New("idempotency reservation changed concurrently")
}

func (s *Store) Complete(ctx context.Context, record idempotency.Record) error {
	count, err := New(s.Pool).CompleteIdempotencyRecord(ctx, CompleteIdempotencyRecordParams{
		ID: record.ID, ResponseKeyVersion: record.KeyVersion, ResponseCiphertext: record.Ciphertext,
	})
	if err != nil {
		return err
	}
	if count != 1 {
		return errors.New("idempotency reservation no longer belongs to this request")
	}
	return nil
}

func (s *Store) PurgeExpired(ctx context.Context, now time.Time) error {
	return New(s.Pool).PurgeIdempotencyRecords(ctx, now)
}

func (s *Store) ListPending(ctx context.Context, tenantID string, version int32, limit int, excluded []int32) ([]idempotency.Record, error) {
	if excluded == nil {
		excluded = []int32{}
	}
	rows, err := New(s.Pool).ListIdempotencyResponsesPendingReencryption(ctx, ListIdempotencyResponsesPendingReencryptionParams{
		TenantID: tenantID, ResponseKeyVersion: version, Limit: int32(limit), ExcludedVersions: excluded, //nolint:gosec // 件数は再暗号化ジョブのバッチ上限以内。
	})
	if err != nil {
		return nil, err
	}
	result := make([]idempotency.Record, 0, len(rows))
	for _, row := range rows {
		result = append(result, fromRow(row))
	}
	return result, nil
}

func (s *Store) ReplaceCipher(ctx context.Context, record idempotency.Record, oldVersion int32) (bool, error) {
	count, err := New(s.Pool).ReplaceIdempotencyResponseCipher(ctx, ReplaceIdempotencyResponseCipherParams{
		ID: record.ID, ResponseKeyVersion: record.KeyVersion, ResponseCiphertext: record.Ciphertext, OldVersion: oldVersion,
	})
	return count == 1, err
}

func (s *Store) PendingCount(ctx context.Context, tenantID string, version int32) (int, error) {
	count, err := New(s.Pool).CountIdempotencyResponsesPendingReencryption(ctx, CountIdempotencyResponsesPendingReencryptionParams{TenantID: tenantID, ResponseKeyVersion: version})
	return int(count), err
}

func fromRow(row *HttpIdempotencyRecord) idempotency.Record {
	return idempotency.Record{
		ID: row.ID, TenantID: row.TenantID, PrincipalID: row.PrincipalID, KeyHash: row.KeyHash,
		Fingerprint: row.Fingerprint, CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt,
		KeyVersion: row.ResponseKeyVersion, Ciphertext: row.ResponseCiphertext,
	}
}
