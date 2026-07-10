package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/ambi/idmagic/internal/shared/spec"
)

// MfaFactorRepository (Authentication)
type MfaFactorRepository struct{ Pool DB }

func (r *MfaFactorRepository) ListBySub(ctx context.Context, sub string) ([]*spec.MfaFactor, error) {
	rows, err := r.Pool.Query(ctx, mfaFactorSelect+" WHERE user_id=$1 ORDER BY created_at", sub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*spec.MfaFactor{}
	for rows.Next() {
		factor, err := scanMfaFactor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, factor)
	}
	return out, rows.Err()
}

func (r *MfaFactorRepository) Find(
	ctx context.Context,
	sub string,
	factorType spec.MfaFactorType,
) (*spec.MfaFactor, error) {
	return scanMfaFactor(r.Pool.QueryRow(ctx, mfaFactorSelect+" WHERE user_id=$1 AND type=$2", sub, factorType))
}

func (r *MfaFactorRepository) Save(ctx context.Context, factor *spec.MfaFactor) error {
	_, err := r.Pool.Exec(ctx, `
INSERT INTO mfa_factors (user_id,type,secret,label,created_at,last_used_at)
VALUES ($1,$2,$3,$4,$5,$6)
ON CONFLICT (user_id,type) DO UPDATE SET secret=EXCLUDED.secret,label=EXCLUDED.label,last_used_at=EXCLUDED.last_used_at,updated_at=now()`,
		factor.UserID, factor.Type, factor.Secret, factor.Label, factor.CreatedAt, factor.LastUsedAt)
	return err
}

func (r *MfaFactorRepository) Delete(ctx context.Context, sub string, factorType spec.MfaFactorType) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mfa_factors WHERE user_id=$1 AND type=$2", sub, factorType)
	return err
}

func (r *MfaFactorRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	_, err := r.Pool.Exec(ctx, "DELETE FROM mfa_factors WHERE user_id=$1", sub)
	return err
}

const mfaFactorSelect = `SELECT user_id,type,secret,label,created_at,last_used_at FROM mfa_factors`

func scanMfaFactor(row RowScanner) (*spec.MfaFactor, error) {
	var factor spec.MfaFactor
	err := row.Scan(&factor.UserID, &factor.Type, &factor.Secret, &factor.Label, &factor.CreatedAt, &factor.LastUsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &factor, factor.Validate()
}
