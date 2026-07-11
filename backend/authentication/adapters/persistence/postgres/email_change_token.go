package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/authentication/adapters/persistence/postgres/sqlcgen"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
)

// EmailChangeTokenStore (Authentication)
type EmailChangeTokenStore struct{ Pool sharedpg.DB }

func (s *EmailChangeTokenStore) Save(
	ctx context.Context,
	record authnports.EmailChangeTokenRecord,
) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlcgen.New(tx)
	if err := queries.DeleteEmailChangeTokensForSub(ctx, record.Sub); err != nil {
		return err
	}
	if err := queries.InsertEmailChangeToken(ctx, sqlcgen.InsertEmailChangeTokenParams{
		TokenHash: record.TokenHash, UserID: record.Sub, NewEmail: record.NewEmail,
		CreatedAt: record.CreatedAt, ExpiresAt: record.ExpiresAt,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *EmailChangeTokenStore) Consume(
	ctx context.Context,
	tokenHash string,
	now time.Time,
) (*authnports.EmailChangeTokenRecord, error) {
	row, err := sqlcgen.New(s.Pool).ConsumeEmailChangeToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	record := &authnports.EmailChangeTokenRecord{
		Sub: row.UserID, TokenHash: row.TokenHash, NewEmail: row.NewEmail,
		CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt,
	}
	if !now.Before(record.ExpiresAt) {
		return nil, nil
	}
	return record, nil
}
