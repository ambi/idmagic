package db_postgres

// AuthEventBucketStore は AuthEventBucketStore (ADR-041 / wi-44) を PostgreSQL に永続化する。
// 攻撃時に個別の AuthenticationFailed を 1 行ずつ書かず、(tenant_id, kind, key_hash, 5 分窓)
// 単位の 1 行へ畳み込む。Record は upsert 1 回で「窓ごとの件数を積む」+「その窓で最初の記録
// だったか」を返し、最初の記録だけが集約イベントを emit する。xmax=0 は当該 upsert が INSERT
// だったこと (= 窓内で最初) を示す PostgreSQL の慣用。

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/db_postgres/sqlcgen"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

const (
	authEventBucketDefaultListLimit = 100
	authEventBucketMaxListLimit     = 1000
)

type AuthEventBucketStore struct{ Pool sharedpg.DB }

func (s *AuthEventBucketStore) queries() *sqlcgen.Queries { return sqlcgen.New(s.Pool) }

func (s *AuthEventBucketStore) Record(
	ctx context.Context,
	kind authnports.AuthEventBucketKind,
	tenantID, keyHash string,
	now time.Time,
) (authnports.AuthEventBucketResult, error) {
	nowUTC := now.UTC()
	windowStart := nowUTC.Truncate(authnports.AuthEventBucketWindow)
	row, err := s.queries().RecordAuthEventBucket(ctx, sqlcgen.RecordAuthEventBucketParams{
		TenantID: tenantID, Kind: string(kind), KeyHash: keyHash, WindowStart: windowStart, FirstSeen: nowUTC,
	})
	if err != nil {
		return authnports.AuthEventBucketResult{}, err
	}
	return authnports.AuthEventBucketResult{
		Bucket: authnports.AuthEventBucket{
			TenantID:    tenantID,
			Kind:        kind,
			KeyHash:     keyHash,
			WindowStart: windowStart,
			Count:       int(row.Count),
			FirstSeen:   row.FirstSeen.UTC(),
			LastSeen:    row.LastSeen.UTC(),
		},
		FirstInWindow: row.Inserted,
	}, nil
}

// DeleteOlderThan は window_start が before より前の bucket を削除し、削除件数を返す
// (ADR-045 の保持期間 sweep / 既定 90 日)。idempotent。
func (s *AuthEventBucketStore) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return s.queries().DeleteAuthEventBucketsOlderThan(ctx, before.UTC())
}

func (s *AuthEventBucketStore) List(
	ctx context.Context,
	tenantID string,
	limit int,
) ([]authnports.AuthEventBucket, error) {
	if limit <= 0 {
		limit = authEventBucketDefaultListLimit
	}
	if limit > authEventBucketMaxListLimit {
		limit = authEventBucketMaxListLimit
	}
	rows, err := s.queries().ListAuthEventBuckets(ctx, sqlcgen.ListAuthEventBucketsParams{
		Column1: tenantID, Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]authnports.AuthEventBucket, 0, len(rows))
	for _, row := range rows {
		out = append(out, authnports.AuthEventBucket{
			TenantID:    row.TenantID,
			Kind:        authnports.AuthEventBucketKind(row.Kind),
			KeyHash:     row.KeyHash,
			WindowStart: row.WindowStart.UTC(),
			Count:       int(row.Count),
			FirstSeen:   row.FirstSeen.UTC(),
			LastSeen:    row.LastSeen.UTC(),
		})
	}
	return out, nil
}
