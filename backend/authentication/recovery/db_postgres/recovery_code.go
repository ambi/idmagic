package db_postgres

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/recovery/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// RecoveryCodeRepository (Authentication) — wi-26 / ADR-087
type RecoveryCodeRepository struct{ Pool sharedpg.DB }

func (r *RecoveryCodeRepository) queries() *Queries { return New(r.Pool) }

func (r *RecoveryCodeRepository) ListBySub(ctx context.Context, sub string) ([]*domain.RecoveryCode, error) {
	rows, err := r.queries().ListRecoveryCodesBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.RecoveryCode, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.RecoveryCode{
			UserID:      row.UserID,
			CodeHash:    row.CodeHash,
			GeneratedAt: row.GeneratedAt,
			ConsumedAt:  timestamptzPtr(row.ConsumedAt),
		})
	}
	return out, nil
}

// ReplaceAll は既存 set を削除してから新しい set を保存する。self-service の再生成専用で
// 単一ユーザーの逐次操作を前提とするため、delete + insert を 1 トランザクションで行う。
func (r *RecoveryCodeRepository) ReplaceAll(
	ctx context.Context,
	sub string,
	codes []*domain.RecoveryCode,
) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := New(tx)
	if err := queries.DeleteRecoveryCodesForSub(ctx, sub); err != nil {
		return err
	}
	for _, c := range codes {
		if err := queries.InsertRecoveryCode(ctx, InsertRecoveryCodeParams{
			UserID: c.UserID, CodeHash: c.CodeHash, GeneratedAt: c.GeneratedAt,
			ConsumedAt: timestamptzOrNil(c.ConsumedAt),
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// MarkConsumed は未使用の code を単一の UPDATE で使用済みにする。consumed_at IS NULL 条件と
// RowsAffected により、競合時も二重消費しない。
func (r *RecoveryCodeRepository) MarkConsumed(
	ctx context.Context,
	sub string,
	codeHash string,
	now time.Time,
) (bool, error) {
	affected, err := r.queries().MarkRecoveryCodeConsumed(ctx, MarkRecoveryCodeConsumedParams{
		UserID: sub, CodeHash: codeHash, ConsumedAt: timestamptzOrNil(&now),
	})
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *RecoveryCodeRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeleteRecoveryCodesForSub(ctx, sub)
}
