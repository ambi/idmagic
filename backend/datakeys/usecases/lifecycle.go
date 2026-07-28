// Package usecases: Layer 4 - Use Cases
package usecases

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// Deps are the dependencies every DataKeys lifecycle usecase shares, mirroring
// backend/signingkeys/usecases.RotateSigningKeyDeps.
type Deps struct {
	Repository ports.DataKeyRepository
	Crypto     envelope_crypto.EnvelopeCrypto
	// Cache is invalidated on Rotate/Disable/Destroy so no worker replica
	// keeps encrypting with a DEK that is no longer active (ADR-148).
	Cache ports.CacheInvalidator
	Emit  func(spec.DomainEvent)
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
