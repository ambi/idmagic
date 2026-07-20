package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/backend/idmanagement/user/db_postgres/sqlcgen"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// EmailChangeTokenStore (IdManagement/User)
type EmailChangeTokenStore struct{ Pool sharedpg.DB }

func (s *EmailChangeTokenStore) Save(
	ctx context.Context,
	record userports.EmailChangeTokenRecord,
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
) (*userports.EmailChangeTokenRecord, error) {
	row, err := sqlcgen.New(s.Pool).ConsumeEmailChangeToken(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	record := &userports.EmailChangeTokenRecord{
		Sub: row.UserID, TokenHash: row.TokenHash, NewEmail: row.NewEmail,
		CreatedAt: row.CreatedAt, ExpiresAt: row.ExpiresAt,
	}
	if !now.Before(record.ExpiresAt) {
		return nil, nil
	}
	return record, nil
}
