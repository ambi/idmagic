package db_postgres

import (
	"context"
	"math"
	"time"

	"github.com/ambi/idmagic/backend/authentication/password/db_postgres/sqlcgen"
	authnports "github.com/ambi/idmagic/backend/authentication/password/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// PasswordHistoryRepository (Authentication)。
type PasswordHistoryRepository struct{ Pool sharedpg.DB }

func (r *PasswordHistoryRepository) queries() *sqlcgen.Queries {
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
	rows, err := r.queries().RecentPasswordHistory(ctx, sqlcgen.RecentPasswordHistoryParams{
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
	return r.queries().InsertPasswordHistory(ctx, sqlcgen.InsertPasswordHistoryParams{
		UserID: sub, Encoded: encoded, CreatedAt: now,
	})
}

func (r *PasswordHistoryRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeletePasswordHistoryForSub(ctx, sub)
}
