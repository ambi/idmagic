package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	authnports "github.com/ambi/idmagic/backend/authentication/password/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// PasswordResetTokenStore (Authentication)
type PasswordResetTokenStore struct{ Pool sharedpg.DB }

func (s *PasswordResetTokenStore) Save(
	ctx context.Context,
	record authnports.PasswordResetTokenRecord,
) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	if err := q.DeletePasswordResetTokensByUser(ctx, record.Sub); err != nil {
		return err
	}
	if err := q.InsertPasswordResetToken(ctx, InsertPasswordResetTokenParams{
		TokenHash: record.TokenHash,
		UserID:    record.Sub,
		CreatedAt: record.CreatedAt,
		ExpiresAt: record.ExpiresAt,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PasswordResetTokenStore) Consume(
	ctx context.Context,
	tokenHash string,
	now time.Time,
) (*authnports.PasswordResetTokenRecord, error) {
	row, err := New(s.Pool).ConsumePasswordResetToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	record := authnports.PasswordResetTokenRecord{
		Sub:       row.UserID,
		TokenHash: row.TokenHash,
		CreatedAt: row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
	}
	if !now.Before(record.ExpiresAt) {
		return nil, nil
	}
	return &record, nil
}
