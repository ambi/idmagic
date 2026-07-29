package db_postgres

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/jackc/pgx/v5/pgtype"
)

// MfaFactorMigratorName is the datakeys/usecases.MigratorRegistry key
// MfaFactorReencryptor registers under (wi-97 T006).
const MfaFactorMigratorName = "mfa_totp_secret"

// MfaFactorReencryptor implements backend/datakeys/ports.FieldMigrator for
// mfa_factors.secret (ADR-148, wi-97 T006): it drives the data_key_reencryption
// job's per-tenant backfill/re-encryption of legacy plaintext rows and rows
// still on a retiring DEK version.
type MfaFactorReencryptor struct {
	Repo *MfaFactorRepository
}

// ReencryptBatch decrypts each pending row (dual-reading legacy plaintext or
// an older DEK version, same as MfaFactorRepository.mfaFactorFromRow) and
// re-encrypts it onto activeVersion, clearing the legacy plaintext column.
// It is idempotent: a row already on activeVersion is excluded by the
// underlying query, so re-running after a partial batch or a crash only
// touches what is still pending.
func (m *MfaFactorReencryptor) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (int, error) {
	ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: tenantID}, "", "")
	rows, err := m.Repo.queries().ListMfaFactorsPendingReencryption(ctx, ListMfaFactorsPendingReencryptionParams{
		TenantID:         tenantID,
		SecretKeyVersion: pgtype.Int4{Int32: int32(activeVersion), Valid: true}, //nolint:gosec // G115: DEK version is a small monotonic counter
		Limit:            int32(batchSize),                                      //nolint:gosec // G115: batchSize is caller-controlled and small
	})
	if err != nil {
		return 0, err
	}

	migrated := 0
	for _, row := range rows {
		factorType := spec.MfaFactorType(row.Type)
		recordID := mfaFactorRecordID(row.UserID, factorType)

		var plaintext string
		switch {
		case row.SecretKeyVersion.Valid && len(row.SecretCiphertext) > 0:
			plaintext, err = m.Repo.Cipher.Decrypt(ctx, tenantID, mfaFactorRecordContext, mfaFactorTable, recordID, mfaFactorSecretField, int(row.SecretKeyVersion.Int32), row.SecretCiphertext)
			if err != nil {
				return migrated, err
			}
		case row.Secret.Valid:
			plaintext = row.Secret.String
		default:
			// Defensive: the query already excludes rows with neither
			// column populated.
			continue
		}

		version, ciphertext, err := m.Repo.Cipher.Encrypt(ctx, tenantID, mfaFactorRecordContext, mfaFactorTable, recordID, mfaFactorSecretField, plaintext)
		if err != nil {
			return migrated, err
		}
		if err := m.Repo.queries().UpdateMfaFactorCiphertext(ctx, UpdateMfaFactorCiphertextParams{
			UserID:           row.UserID,
			Type:             row.Type,
			SecretKeyVersion: pgtype.Int4{Int32: int32(version), Valid: true}, //nolint:gosec // G115: DEK version is a small monotonic counter
			SecretCiphertext: ciphertext,
		}); err != nil {
			return migrated, err
		}
		migrated++
	}
	return migrated, nil
}

// PendingCount reports how many of tenantID's rows still carry secret
// material not yet on activeVersion, without migrating anything (the
// verification query DestroyTenantDataKey's gate uses).
func (m *MfaFactorReencryptor) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: tenantID}, "", "")
	count, err := m.Repo.queries().CountMfaFactorsPendingReencryption(ctx, CountMfaFactorsPendingReencryptionParams{
		TenantID:         tenantID,
		SecretKeyVersion: pgtype.Int4{Int32: int32(activeVersion), Valid: true}, //nolint:gosec // G115: DEK version is a small monotonic counter
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}
