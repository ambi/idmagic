// Package ports: Layer 2 - Ports
package ports

import "context"

// FieldMigrator re-encrypts one owning module's envelope-encrypted field
// batch-wise onto a tenant's current active DataEncryptionKey version
// (docs/modules/data-keys/, wi-97 T006). Implementations live
// in the owning module's adapter (e.g.
// backend/authentication/totp/db_postgres.MfaFactorReencryptor) and are
// registered by name with usecases.MigratorRegistry at bootstrap, so DataKeys
// never depends on any consumer's schema — mirroring how backend/jobs'
// HandlerRegistry lets consumer contexts extend a shared job kind vocabulary
// without Jobs knowing their business logic.
type FieldMigrator interface {
	// ReencryptBatch は activeVersion へ未移行の行を batchSize 件まで移す。
	// excludedVersions の行は上限の適用前に除外し、変更しない。
	// すでに activeVersion の行も再選択せず、部分的な実行の後でも安全に再開できる。
	ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int, excludedVersions []int32) (migrated int, err error)

	// PendingCount reports how many of tenantID's rows are not yet on
	// activeVersion, without migrating anything. It is the read-only
	// verification query DestroyTenantDataKey's gate uses: destroying any
	// non-active version while this is > 0 could erase the only key able to
	// decrypt a still-referenced row, so the gate is conservative and blocks
	// on any staleness rather than checking a specific old version.
	// 再暗号化から除外した無効化済みと破棄済みのバージョンの行も数える。
	PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error)
}
