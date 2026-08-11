// Package ports: Layer 2 - Ports
package ports

import "context"

// FieldMigrator re-encrypts one owning context's envelope-encrypted field
// batch-wise onto a tenant's current active DataEncryptionKey version
// (spec/contexts/data-keys.yaml, wi-97 T006). Implementations live
// in the owning context's adapter (e.g.
// backend/authentication/totp/db_postgres.MfaFactorReencryptor) and are
// registered by name with usecases.MigratorRegistry at bootstrap, so DataKeys
// never depends on any consumer's schema — mirroring how backend/jobs'
// HandlerRegistry lets consumer contexts extend a shared job kind vocabulary
// without Jobs knowing their business logic.
type FieldMigrator interface {
	// ReencryptBatch re-encrypts up to batchSize of tenantID's rows that are
	// not yet on activeVersion — legacy plaintext or an older DEK version —
	// and returns how many it migrated. Implementations must be idempotent
	// (JobHandlerIdempotency): a row already on activeVersion is simply not
	// reselected, so calling this again after a partial run or a crash just
	// continues rather than reprocessing.
	ReencryptBatch(ctx context.Context, tenantID string, activeVersion, batchSize int) (migrated int, err error)

	// PendingCount reports how many of tenantID's rows are not yet on
	// activeVersion, without migrating anything. It is the read-only
	// verification query DestroyTenantDataKey's gate uses: destroying any
	// non-active version while this is > 0 could erase the only key able to
	// decrypt a still-referenced row, so the gate is conservative and blocks
	// on any staleness rather than checking a specific old version.
	PendingCount(ctx context.Context, tenantID string, activeVersion int) (int, error)
}
