package db_postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// EmailChangeTokenStore (IdManagement/User)
type EmailChangeTokenStore struct{ Pool sharedpg.DB }

var _ userports.EmailChangeTokenStore = (*EmailChangeTokenStore)(nil)

func (s *EmailChangeTokenStore) Save(ctx context.Context, envelope actiontoken.Envelope) error {
	if envelope.Purpose != actiontoken.PurposeEmailChange {
		return actiontoken.ErrPurposeMismatch
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := New(tx)
	// 一人につき生きている確認トークンは一つだけにする。
	if err := queries.DeleteEmailChangeTokensForSub(ctx, envelope.Subject); err != nil {
		return err
	}
	if err := queries.InsertEmailChangeToken(ctx, InsertEmailChangeTokenParams{
		TokenHash: string(envelope.Digest),
		ID:        envelope.ID,
		UserID:    envelope.Subject,
		Purpose:   string(envelope.Purpose),
		NewEmail:  envelope.Payload[userports.PayloadKeyNewEmail],
		CreatedAt: envelope.IssuedAt,
		ExpiresAt: envelope.ExpiresAt,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *EmailChangeTokenStore) Find(
	ctx context.Context,
	digest actiontoken.Digest,
) (*actiontoken.Envelope, error) {
	row, err := New(s.Pool).FindUnusedEmailChangeToken(ctx, string(digest))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	purpose, err := actiontoken.ParsePurpose(row.Purpose)
	if err != nil {
		return nil, err
	}
	return &actiontoken.Envelope{
		ID:        row.ID,
		Purpose:   purpose,
		Subject:   row.UserID,
		Payload:   actiontoken.Payload{userports.PayloadKeyNewEmail: row.NewEmail},
		IssuedAt:  row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
		Digest:    actiontoken.Digest(row.TokenHash),
	}, nil
}

// ConsumeAndApply は使用済み化と primary email の確定を一つのトランザクションで
// 行う。どちらかが失敗すれば ROLLBACK が全体を戻すので、トークンは未使用のまま残る。
func (s *EmailChangeTokenStore) ConsumeAndApply(
	ctx context.Context,
	commit userports.EmailChangeCommit,
) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// 未使用の行に限って使用済みにする。並行する二つの確定要求のうち、行を掴めた
	// 一方だけが作用へ進む。
	if _, err := New(tx).MarkEmailChangeTokenUsed(ctx, MarkEmailChangeTokenUsedParams{
		UsedAt: commit.Now, TokenHash: string(commit.Digest),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return actiontoken.ErrAlreadyConsumed
		}
		return err
	}
	if err := SaveUserTx(ctx, tx, commit.User); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
