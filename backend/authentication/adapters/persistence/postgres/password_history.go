package postgres

import (
	"context"
	"math"
	"time"

	"github.com/ambi/idmagic/backend/authentication/adapters/persistence/postgres/sqlcgen"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// PasswordHistoryRepository (Authentication)。ctx が pgx.Tx を運ぶ場合
// (wi-184 T003 の transaction runner 経由) はその tx を使い、業務更新と
// event_logs 追記を同一 commit にする (ADR-094)。
type PasswordHistoryRepository struct{ Pool sharedpg.DB }

func (r *PasswordHistoryRepository) queries(ctx context.Context) *sqlcgen.Queries {
	if tx, ok := sharedpg.TxFromContext(ctx); ok {
		return sqlcgen.New(tx)
	}
	return sqlcgen.New(r.Pool)
}

func (r *PasswordHistoryRepository) Recent(
	ctx context.Context,
	sub string,
	depth int,
) ([]authnports.PasswordHistoryEntry, error) {
	if depth <= 0 {
		return nil, nil
	}
	if depth > math.MaxInt32 {
		depth = math.MaxInt32
	}
	rows, err := r.queries(ctx).RecentPasswordHistory(ctx, sqlcgen.RecentPasswordHistoryParams{
		UserID: sub, Limit: int32(depth),
	})
	if err != nil {
		return nil, err
	}
	out := make([]authnports.PasswordHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, authnports.PasswordHistoryEntry{Encoded: row.Encoded, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *PasswordHistoryRepository) Add(ctx context.Context, sub, encoded string, now time.Time) error {
	return r.queries(ctx).InsertPasswordHistory(ctx, sqlcgen.InsertPasswordHistoryParams{
		UserID: sub, Encoded: encoded, CreatedAt: now,
	})
}

func (r *PasswordHistoryRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries(ctx).DeletePasswordHistoryForSub(ctx, sub)
}
