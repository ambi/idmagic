package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/authentication/totp/adapters/persistence/postgres/sqlcgen"
	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// MfaFactorRepository (Authentication)
type MfaFactorRepository struct{ Pool sharedpg.DB }

func (r *MfaFactorRepository) queries() *sqlcgen.Queries { return sqlcgen.New(r.Pool) }

func (r *MfaFactorRepository) ListBySub(ctx context.Context, sub string) ([]*domain.MfaFactor, error) {
	rows, err := r.queries().ListMfaFactorsBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.MfaFactor, 0, len(rows))
	for _, row := range rows {
		factor, err := mfaFactorFromRow(row.UserID, row.Type, row.Secret, row.Label, row.CreatedAt, row.LastUsedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, factor)
	}
	return out, nil
}

func (r *MfaFactorRepository) Find(
	ctx context.Context,
	sub string,
	factorType spec.MfaFactorType,
) (*domain.MfaFactor, error) {
	row, err := r.queries().GetMfaFactor(ctx, sqlcgen.GetMfaFactorParams{UserID: sub, Type: string(factorType)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return mfaFactorFromRow(row.UserID, row.Type, row.Secret, row.Label, row.CreatedAt, row.LastUsedAt)
}

func (r *MfaFactorRepository) Save(ctx context.Context, factor *domain.MfaFactor) error {
	return r.queries().UpsertMfaFactor(ctx, sqlcgen.UpsertMfaFactorParams{
		UserID:     factor.UserID,
		Type:       string(factor.Type),
		Secret:     textOrNil(factor.Secret),
		Label:      textOrNil(factor.Label),
		CreatedAt:  factor.CreatedAt,
		LastUsedAt: timestamptzOrNil(factor.LastUsedAt),
	})
}

func (r *MfaFactorRepository) Delete(ctx context.Context, sub string, factorType spec.MfaFactorType) error {
	return r.queries().DeleteMfaFactor(ctx, sqlcgen.DeleteMfaFactorParams{UserID: sub, Type: string(factorType)})
}

func (r *MfaFactorRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeleteMfaFactorsForSub(ctx, sub)
}

func mfaFactorFromRow(
	userID, factorType string,
	secret, label pgtype.Text,
	createdAt time.Time,
	lastUsedAt pgtype.Timestamptz,
) (*domain.MfaFactor, error) {
	factor := &domain.MfaFactor{
		UserID:     userID,
		Type:       spec.MfaFactorType(factorType),
		Secret:     textPtr(secret),
		Label:      textPtr(label),
		CreatedAt:  createdAt,
		LastUsedAt: timestamptzPtr(lastUsedAt),
	}
	return factor, factor.Validate()
}
