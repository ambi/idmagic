package db_postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	authnports "github.com/ambi/idmagic/backend/authentication/password/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/security/actiontoken"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// PasswordResetTokenStore (Authentication)
type PasswordResetTokenStore struct {
	pool      sharedpg.DB
	usersInTx func(pgx.Tx) userports.UserWriter
}

// NewPasswordResetTokenStore は、使用済み化と同じトランザクションで User を保存するため、
// IdManagement の UserWriter を作る usersInTx を受け取る。
func NewPasswordResetTokenStore(pool sharedpg.DB, usersInTx func(pgx.Tx) userports.UserWriter) *PasswordResetTokenStore {
	return &PasswordResetTokenStore{pool: pool, usersInTx: usersInTx}
}

var _ authnports.PasswordResetTokenStore = (*PasswordResetTokenStore)(nil)

func (s *PasswordResetTokenStore) Save(ctx context.Context, envelope actiontoken.Envelope) error {
	if envelope.Purpose != actiontoken.PurposePasswordReset {
		return actiontoken.ErrPurposeMismatch
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	// 一人につき生きているリセットトークンは一つだけにする。
	if err := q.DeletePasswordResetTokensByUser(ctx, envelope.Subject); err != nil {
		return err
	}
	if err := q.InsertPasswordResetToken(ctx, InsertPasswordResetTokenParams{
		TokenHash: string(envelope.Digest),
		ID:        envelope.ID,
		UserID:    envelope.Subject,
		Purpose:   string(envelope.Purpose),
		CreatedAt: envelope.IssuedAt,
		ExpiresAt: envelope.ExpiresAt,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *PasswordResetTokenStore) Find(
	ctx context.Context,
	digest actiontoken.Digest,
) (*actiontoken.Envelope, error) {
	row, err := New(s.pool).FindUnusedPasswordResetToken(ctx, string(digest))
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
		IssuedAt:  row.CreatedAt,
		ExpiresAt: row.ExpiresAt,
		Digest:    actiontoken.Digest(row.TokenHash),
	}, nil
}

// ConsumeAndApply は使用済み化、User の保存、パスワード履歴の追加を一つの
// トランザクションで確定する。どれかが失敗すれば ROLLBACK が全体を戻すので、
// トークンは未使用のまま残り、同じトークンで作用が二回成功する状態を作らない。
func (s *PasswordResetTokenStore) ConsumeAndApply(
	ctx context.Context,
	commit authnports.PasswordResetCommit,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := New(tx)
	// 未使用の行に限って使用済みにする。並行する二つの確定要求のうち、行を掴めた
	// 一方だけが作用へ進む。
	if _, err := q.MarkPasswordResetTokenUsed(ctx, MarkPasswordResetTokenUsedParams{
		UsedAt: commit.Now, TokenHash: string(commit.Digest),
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return actiontoken.ErrAlreadyConsumed
		}
		return err
	}
	updated, err := s.usersInTx(tx).Update(ctx, commit.UserID, commit.Apply)
	if err != nil {
		return err
	}
	if updated == nil {
		return authnports.ErrResetUserMissing
	}
	if err := q.InsertPasswordHistory(ctx, InsertPasswordHistoryParams{
		UserID: commit.UserID, Encoded: commit.PasswordEncoded, CreatedAt: commit.Now,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
