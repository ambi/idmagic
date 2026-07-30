package db_postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"

	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
)

// IdentityProviderSecretMigratorName registers ConnectionSecretReencryptor with
// backend/datakeys' MigratorRegistry (ADR-150, mirrors totppostgres.MfaFactorMigratorName).
const IdentityProviderSecretMigratorName = "identity_provider_client_secret"

// ConnectionSecretReencryptor migrates identity_provider_connections.secret_reference off the
// env: scheme and onto envelope encryption (ADR-150). Unlike the MFA reencryptor it migrates
// from a legacy *reference* rather than legacy *plaintext*, so a row can genuinely fail to
// migrate (its env: variable is no longer set): ReencryptBatch skips such rows rather than
// failing the whole batch, leaving them counted by PendingCount until an operator re-enters the
// secret through the admin UI (which encrypts it directly on the next Save).
type ConnectionSecretReencryptor struct {
	Repo        *ConnectionRepository
	EnvResolver federationports.SecretResolver
}

func (m *ConnectionSecretReencryptor) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (int, error) {
	rows, err := New(m.Repo.Pool).ListIdentityProviderConnectionsPendingSecretReencryption(ctx, ListIdentityProviderConnectionsPendingSecretReencryptionParams{
		TenantID:         tenantID,
		SecretKeyVersion: pgtype.Int4{Int32: int32(activeVersion), Valid: true}, //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
		Limit:            int32(batchSize),                                      //nolint:gosec // G115: batch size is caller-controlled and small
	})
	if err != nil {
		return 0, err
	}
	migrated := 0
	for _, row := range rows {
		plaintext, ok, err := m.resolvePlaintext(ctx, tenantID, row)
		if err != nil {
			return migrated, err
		}
		if !ok {
			continue // unresolvable env: reference (or missing dependency): leave pending (ADR-150)
		}
		version, ciphertext, err := m.Repo.Cipher.Encrypt(
			ctx, tenantID, federationSecretRecordContext, federationSecretTable,
			connectionRecordID(tenantID, row.ProviderID), federationSecretField, plaintext,
		)
		if err != nil {
			return migrated, err
		}
		if err := New(m.Repo.Pool).UpdateIdentityProviderConnectionSecretCiphertext(ctx, UpdateIdentityProviderConnectionSecretCiphertextParams{
			TenantID:         tenantID,
			ProviderID:       row.ProviderID,
			SecretKeyVersion: pgtype.Int4{Int32: int32(version), Valid: true}, //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
			SecretCiphertext: ciphertext,
		}); err != nil {
			return migrated, err
		}
		migrated++
	}
	return migrated, nil
}

func (m *ConnectionSecretReencryptor) resolvePlaintext(
	ctx context.Context, tenantID string, row *ListIdentityProviderConnectionsPendingSecretReencryptionRow,
) (plaintext string, ok bool, err error) {
	if row.SecretKeyVersion.Valid && len(row.SecretCiphertext) > 0 {
		plaintext, err = m.Repo.Cipher.Decrypt(
			ctx, tenantID, federationSecretRecordContext, federationSecretTable,
			connectionRecordID(tenantID, row.ProviderID), federationSecretField,
			int(row.SecretKeyVersion.Int32), row.SecretCiphertext,
		)
		if err != nil {
			return "", false, err
		}
		return plaintext, true, nil
	}
	if !row.SecretReference.Valid || row.SecretReference.String == "" || m.EnvResolver == nil {
		return "", false, nil
	}
	plaintext, err = m.EnvResolver.Resolve(ctx, row.SecretReference.String)
	if err != nil {
		return "", false, nil //nolint:nilerr // unresolvable env: reference is left pending, not a batch failure
	}
	return plaintext, true, nil
}

func (m *ConnectionSecretReencryptor) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	count, err := New(m.Repo.Pool).CountIdentityProviderConnectionsPendingSecretReencryption(ctx, CountIdentityProviderConnectionsPendingSecretReencryptionParams{
		TenantID:         tenantID,
		SecretKeyVersion: pgtype.Int4{Int32: int32(activeVersion), Valid: true}, //nolint:gosec // G115: DEK version is a small monotonic counter, well under int32 max
	})
	return int(count), err
}
