package db_postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// DataKeys の MigratorRegistry へ登録する名前。
const (
	ConnectorSecretMigratorName = "token_vault_connector_secret"
	ConnectionTokenMigratorName = "token_vault_connection_tokens"
)

// ConnectorSecretReencryptor はコネクタのシークレットを有効な DEK へ移す。
type ConnectorSecretReencryptor struct{ Repo *ConnectorRepository }

// ReencryptBatch は読んだ版のままの行だけを書き換える。読んだ後に更新された行の新しい値を残す。
func (m *ConnectorSecretReencryptor) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (int, error) {
	rows, err := New(m.Repo.Pool).ListConnectorsPendingReencryption(ctx, ListConnectorsPendingReencryptionParams{
		TenantID: tenantID, SecretKeyVersion: int32(activeVersion), Limit: int32(batchSize), //nolint:gosec // G115: small values
	})
	if err != nil {
		return 0, err
	}
	migrated := 0
	for _, row := range rows {
		secret, err := m.Repo.Cipher.Decrypt(ctx, tenantID, recordContext, connectorsTable, row.ID, secretField, int(row.SecretKeyVersion), row.SecretCiphertext)
		if err != nil {
			return migrated, err
		}
		version, ciphertext, err := m.Repo.Cipher.Encrypt(ctx, tenantID, recordContext, connectorsTable, row.ID, secretField, secret)
		if err != nil {
			return migrated, err
		}
		updated, err := New(m.Repo.Pool).ReencryptConnectorSecret(ctx, ReencryptConnectorSecretParams{
			TenantID: tenantID, ID: row.ID, SecretKeyVersion: int32(version), SecretCiphertext: ciphertext, //nolint:gosec // G115: small values
			Version: row.Version, SecretKeyVersion_2: row.SecretKeyVersion,
		})
		if err != nil {
			return migrated, err
		}
		migrated += int(updated)
	}
	return migrated, nil
}

// PendingCount は未移行の件数を返す。
func (m *ConnectorSecretReencryptor) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	count, err := New(m.Repo.Pool).CountConnectorsPendingReencryption(ctx, CountConnectorsPendingReencryptionParams{
		TenantID: tenantID, SecretKeyVersion: int32(activeVersion), //nolint:gosec // G115: small values
	})
	return int(count), err
}

// ConnectionTokenReencryptor は接続のトークンを有効な DEK へ移す。
type ConnectionTokenReencryptor struct{ Repo *ConnectionRepository }

// ReencryptBatch はトークンの版が読んだ値のままの行だけを書き換える。
func (m *ConnectionTokenReencryptor) ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (int, error) {
	active := pgtype.Int4{Int32: int32(activeVersion), Valid: true} //nolint:gosec // G115: small values
	rows, err := New(m.Repo.Pool).ListConnectionsPendingReencryption(ctx, ListConnectionsPendingReencryptionParams{
		TenantID: tenantID, TokenKeyVersion: active, Limit: int32(batchSize), //nolint:gosec // G115: small values
	})
	if err != nil {
		return 0, err
	}
	migrated := 0
	for _, row := range rows {
		connection, err := m.Repo.toDomain(ctx, row)
		if err != nil {
			return migrated, err
		}
		sealed, err := seal(ctx, m.Repo.Cipher, tenantID, connectionRecordID(row.ConnectorID, row.UserID), connection.AccessToken, connection.RefreshToken)
		if err != nil {
			return migrated, err
		}
		updated, err := New(m.Repo.Pool).ReencryptConnectionTokens(ctx, ReencryptConnectionTokensParams{
			TenantID: tenantID, ID: row.ID, TokenKeyVersion: sealed.version, AccessTokenCiphertext: sealed.access,
			RefreshTokenCiphertext: sealed.refresh, TokenVersion: row.TokenVersion, TokenKeyVersion_2: row.TokenKeyVersion,
		})
		if err != nil {
			return migrated, err
		}
		migrated += int(updated)
	}
	return migrated, nil
}

// PendingCount は未移行の件数を返す。
func (m *ConnectionTokenReencryptor) PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error) {
	count, err := New(m.Repo.Pool).CountConnectionsPendingReencryption(ctx, CountConnectionsPendingReencryptionParams{
		TenantID: tenantID, TokenKeyVersion: pgtype.Int4{Int32: int32(activeVersion), Valid: true}, //nolint:gosec // G115: small values
	})
	return int(count), err
}
