package db_postgres

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/authentication/totp/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	mfaFactorRecordContext = "Authentication"
	mfaFactorTable         = "mfa_factors"
	mfaFactorSecretField   = "secret"
)

// MfaFactorRepository (Authentication). Cipher envelope-encrypts Secret at
// rest: Save always encrypts and never writes the legacy plaintext
// column; Find/ListBySub dual-read rows written before this migration
// (secret populated, secret_ciphertext NULL) until wi-97 T006 backfills them.
type MfaFactorRepository struct {
	Pool   sharedpg.DB
	Cipher ports.SecretCipher
}

func (r *MfaFactorRepository) queries() *Queries { return New(r.Pool) }

func (r *MfaFactorRepository) ListBySub(ctx context.Context, sub string) ([]*domain.MfaFactor, error) {
	rows, err := r.queries().ListMfaFactorsBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.MfaFactor, 0, len(rows))
	for _, row := range rows {
		factor, err := r.mfaFactorFromRow(ctx, row.UserID, row.Type, row.Secret, row.SecretKeyVersion, row.SecretCiphertext, row.Label, row.CreatedAt, row.LastUsedAt)
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
	row, err := r.queries().GetMfaFactor(ctx, GetMfaFactorParams{UserID: sub, Type: string(factorType)})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r.mfaFactorFromRow(ctx, row.UserID, row.Type, row.Secret, row.SecretKeyVersion, row.SecretCiphertext, row.Label, row.CreatedAt, row.LastUsedAt)
}

func (r *MfaFactorRepository) Save(ctx context.Context, factor *domain.MfaFactor) error {
	params := UpsertMfaFactorParams{
		UserID:     factor.UserID,
		Type:       string(factor.Type),
		Label:      textOrNil(factor.Label),
		CreatedAt:  factor.CreatedAt,
		LastUsedAt: timestamptzOrNil(factor.LastUsedAt),
	}
	if factor.Secret != nil {
		version, ciphertext, err := r.Cipher.Encrypt(
			ctx, tenancy.TenantID(ctx), mfaFactorRecordContext, mfaFactorTable,
			mfaFactorRecordID(factor.UserID, factor.Type), mfaFactorSecretField, *factor.Secret,
		)
		if err != nil {
			return err
		}
		params.SecretKeyVersion = pgtype.Int4{Int32: int32(version), Valid: true} //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
		params.SecretCiphertext = ciphertext
	}
	return r.queries().UpsertMfaFactor(ctx, params)
}

func (r *MfaFactorRepository) Delete(ctx context.Context, sub string, factorType spec.MfaFactorType) error {
	return r.queries().DeleteMfaFactor(ctx, DeleteMfaFactorParams{UserID: sub, Type: string(factorType)})
}

func (r *MfaFactorRepository) DeleteAllForSub(ctx context.Context, sub string) error {
	return r.queries().DeleteMfaFactorsForSub(ctx, sub)
}

func (r *MfaFactorRepository) mfaFactorFromRow(
	ctx context.Context,
	userID, factorType string,
	secret pgtype.Text,
	secretKeyVersion pgtype.Int4,
	secretCiphertext []byte,
	label pgtype.Text,
	createdAt time.Time,
	lastUsedAt pgtype.Timestamptz,
) (*domain.MfaFactor, error) {
	factor := &domain.MfaFactor{
		UserID:     userID,
		Type:       spec.MfaFactorType(factorType),
		Label:      textPtr(label),
		CreatedAt:  createdAt,
		LastUsedAt: timestamptzPtr(lastUsedAt),
	}
	switch {
	case secretKeyVersion.Valid && len(secretCiphertext) > 0:
		plaintext, err := r.Cipher.Decrypt(
			ctx, tenancy.TenantID(ctx), mfaFactorRecordContext, mfaFactorTable,
			mfaFactorRecordID(userID, spec.MfaFactorType(factorType)), mfaFactorSecretField,
			int(secretKeyVersion.Int32), secretCiphertext,
		)
		if err != nil {
			return nil, err
		}
		factor.Secret = &plaintext
	case secret.Valid:
		value := secret.String
		factor.Secret = &value
	}
	return factor, factor.Validate()
}

func mfaFactorRecordID(userID string, factorType spec.MfaFactorType) string {
	return userID + ":" + string(factorType)
}
