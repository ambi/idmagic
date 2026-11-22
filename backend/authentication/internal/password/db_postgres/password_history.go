package db_postgres

import (
	"context"
	"math"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	"github.com/jackc/pgx/v5"
)

// PasswordHistoryRepository (Authentication)。
type PasswordHistoryRepository struct{ Pool DBTX }

// PasswordHistoryWriterInTx は、User の保存と同じ Tx へ履歴を追加する操作を返す。
func PasswordHistoryWriterInTx(tx pgx.Tx) passwordports.PasswordHistoryWriter {
	if tx == nil {
		panic("authentication: PasswordHistoryWriterInTx requires a transaction")
	}
	return &PasswordHistoryRepository{Pool: tx}
}

func (r *PasswordHistoryRepository) queries() *Queries {
	return New(r.Pool)
}

func (r *PasswordHistoryRepository) Recent(
	ctx context.Context,
	sub string,
	depth int,
) ([]passwordports.PasswordHistoryEntry, error) {
	if depth <= 0 {
		return nil, nil
	}
	if depth > math.MaxInt32 {
		depth = math.MaxInt32
	}
	rows, err := r.queries().RecentPasswordHistory(ctx, RecentPasswordHistoryParams{
		UserID: sub, Limit: int32(depth),
	})
	if err != nil {
		return nil, err
	}
	out := make([]passwordports.PasswordHistoryEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, passwordports.PasswordHistoryEntry{Encoded: row.Encoded, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *PasswordHistoryRepository) Add(ctx context.Context, sub, encoded string, now time.Time) error {
	return r.queries().InsertPasswordHistory(ctx, InsertPasswordHistoryParams{
		UserID: sub, Encoded: encoded, CreatedAt: now,
	})
}

func (r *PasswordHistoryRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeletePasswordHistoryForSub(ctx, sub)
}
