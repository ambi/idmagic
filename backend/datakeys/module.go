// Package datakeys composes the DataKeys bounded context (ADR-148).
package datakeys

import (
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

// Module holds the dependencies other contexts' repositories (e.g.
// Authentication's MFA factor store, wi-97 T005) need to envelope-encrypt
// reversible secrets, plus what the system_admin health endpoint
// (ListTenantDataKeyHealth, wi-97 T007) needs.
type Module struct {
	Repository ports.DataKeyRepository
	Cache      *usecases.DataKeyCache
	// Crypto is the same EnvelopeCrypto instance Cache wraps; the health
	// endpoint uses it directly for Healthy()/Provider() (wi-97 T007), which
	// have no reason to go through the per-tenant DEK cache.
	Crypto envelope_crypto.EnvelopeCrypto
	// Migrators lists every owning context's registered FieldMigrator
	// (wi-97 T006): the data_key_reencryption job drives them, Rotate
	// enqueues per registered name, and Destroy's gate checks each one's
	// PendingCount. Bootstrap constructs one instance shared across
	// RunWorker's handler registration and the lifecycle usecases.Deps
	// passed wherever Rotate/Destroy are called.
	Migrators *usecases.MigratorRegistry
}
