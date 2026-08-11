// Package usecases: Layer 4 - Use Cases
package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Deps are the dependencies every DataKeys lifecycle usecase shares, mirroring
// backend/signingkeys/usecases.RotateSigningKeyDeps.
type Deps struct {
	Repository ports.DataKeyRepository
	Crypto     envelope_crypto.EnvelopeCrypto
	// Cache is invalidated on Rotate/Disable/Destroy so no worker replica
	// keeps encrypting with a DEK that is no longer active.
	Cache ports.CacheInvalidator
	Emit  func(spec.DomainEvent)
	// Migrators lists every owning context's registered FieldMigrator
	// (wi-97 T006): Rotate enqueues a data_key_reencryption Job per name, and
	// Destroy refuses to erase a wrapped_dek while any migrator still
	// reports pending rows. nil skips both (wiring gaps in tests/tools that
	// don't exercise rotation/destroy); production bootstrap always sets it.
	Migrators *MigratorRegistry
	// Jobs enqueues the data_key_reencryption Job that Rotate triggers. nil
	// skips enqueueing, same rationale as Migrators.
	Jobs jobsports.JobRepository
}

func BootstrapTenantDataKey(ctx context.Context, deps Deps, tenantID string, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	plaintextDEK, err := deps.Crypto.GenerateDataKey(ctx)
	if err != nil {
		return nil, err
	}
	wrapped, masterKeyID, err := deps.Crypto.Wrap(ctx, tenantID, plaintextDEK)
	if err != nil {
		return nil, err
	}
	key, err := deps.Repository.Bootstrap(ctx, tenantID, wrapped, masterKeyID, now)
	if err != nil {
		return nil, err
	}
	if deps.Emit != nil {
		deps.Emit(&domain.DataEncryptionKeyBootstrapped{At: now, TenantID: tenantID, Version: key.Version})
	}
	return key, nil
}

func RotateTenantDataKey(ctx context.Context, deps Deps, tenantID string, now time.Time) (*domain.TenantDataEncryptionKey, error) {
	plaintextDEK, err := deps.Crypto.GenerateDataKey(ctx)
	if err != nil {
		return nil, err
	}
	wrapped, masterKeyID, err := deps.Crypto.Wrap(ctx, tenantID, plaintextDEK)
	if err != nil {
		return nil, err
	}
	next, previous, err := deps.Repository.Rotate(ctx, tenantID, wrapped, masterKeyID, now)
	if err != nil {
		return nil, err
	}
	if deps.Cache != nil {
		deps.Cache.Invalidate(tenantID)
	}
	if deps.Emit != nil {
		deps.Emit(&domain.DataEncryptionKeyRotated{At: now, TenantID: tenantID, PreviousVersion: previous.Version, NewVersion: next.Version})
	}
	if deps.Migrators != nil && deps.Jobs != nil {
		for _, name := range deps.Migrators.Names() {
			if enqErr := EnqueueReencryptionJob(ctx, deps.Jobs, tenantID, name, now); enqErr != nil {
				// Not fatal to the rotation itself, which already committed:
				// the idmagic-batch reencryption sweep is the fallback that
				// still catches this tenant/migrator later.
				logging.Warn(ctx, "datakeys: failed to enqueue reencryption job after rotate", "error", enqErr, "tenant_id", tenantID, "migrator", name)
			}
		}
	}
	return next, nil
}

func DisableTenantDataKey(ctx context.Context, deps Deps, tenantID string, version int, now time.Time) error {
	_, err := deps.Repository.Disable(ctx, tenantID, version, now)
	if err != nil {
		return err
	}
	if deps.Cache != nil {
		deps.Cache.Invalidate(tenantID)
	}
	if deps.Emit != nil {
		deps.Emit(&domain.DataEncryptionKeyDisabled{At: now, TenantID: tenantID, Version: version})
	}
	return nil
}

func DestroyTenantDataKey(ctx context.Context, deps Deps, tenantID string, version int, now time.Time) error {
	if deps.Migrators != nil {
		active, err := deps.Repository.FindActive(ctx, tenantID)
		if err != nil {
			return err
		}
		for _, name := range deps.Migrators.Names() {
			migrator, ok := deps.Migrators.Lookup(name)
			if !ok {
				continue
			}
			pending, pendingErr := migrator.PendingCount(ctx, tenantID, active.Version)
			if pendingErr != nil {
				return pendingErr
			}
			if pending > 0 {
				return fmt.Errorf("%w: %d record(s) in %q not yet re-encrypted onto active version %d", domain.ErrDataKeyStillReferenced, pending, name, active.Version)
			}
		}
	}
	_, err := deps.Repository.Destroy(ctx, tenantID, version, now)
	if err != nil {
		return err
	}
	if deps.Cache != nil {
		deps.Cache.Invalidate(tenantID)
	}
	if deps.Emit != nil {
		deps.Emit(&domain.DataEncryptionKeyDestroyed{At: now, TenantID: tenantID, Version: version})
	}
	return nil
}
