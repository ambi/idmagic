package postgres

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// RecoveryCodeRepository (Authentication) — wi-26 / ADR-087
type RecoveryCodeRepository struct{ Pool DB }

const recoveryCodeSelect = `SELECT user_id,code_hash,generated_at,consumed_at FROM recovery_codes`

func (r *RecoveryCodeRepository) ListBySub(ctx context.Context, sub string) ([]*spec.RecoveryCode, error) {
	rows, err := r.Pool.Query(ctx, recoveryCodeSelect+" WHERE user_id=$1 ORDER BY generated_at", sub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*spec.RecoveryCode{}
	for rows.Next() {
		var c spec.RecoveryCode
		if err := rows.Scan(&c.UserID, &c.CodeHash, &c.GeneratedAt, &c.ConsumedAt); err != nil {
			return nil, err
		}
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ReplaceAll は既存 set を削除してから新しい set を保存する。self-service の再生成専用で
// 単一ユーザーの逐次操作を前提とするため、delete + insert の 2 文で構成する。
func (r *RecoveryCodeRepository) ReplaceAll(
	ctx context.Context,
	sub string,
	codes []*spec.RecoveryCode,
) error {
	if _, err := r.Pool.Exec(ctx, "DELETE FROM recovery_codes WHERE user_id=$1", sub); err != nil {
		return err
	}
	for _, c := range codes {
		if _, err := r.Pool.Exec(ctx,
			"INSERT INTO recovery_codes (user_id,code_hash,generated_at,consumed_at) VALUES ($1,$2,$3,$4)",
			c.UserID, c.CodeHash, c.GeneratedAt, c.ConsumedAt); err != nil {
			return err
		}
	}
	return nil
}

// MarkConsumed は未使用の code を単一の UPDATE で使用済みにする。consumed_at IS NULL 条件と
// RowsAffected により、競合時も二重消費しない。
func (r *RecoveryCodeRepository) MarkConsumed(
	ctx context.Context,
	sub string,
	codeHash string,
	now time.Time,
) (bool, error) {
	tag, err := r.Pool.Exec(ctx,
		"UPDATE recovery_codes SET consumed_at=$3 WHERE user_id=$1 AND code_hash=$2 AND consumed_at IS NULL",
		sub, codeHash, now)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *RecoveryCodeRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM recovery_codes WHERE user_id=$1", sub)
	return err
}
