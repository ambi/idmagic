// Package ports: Layer 2 - Ports
package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
)

// DataKeyRepository owns the atomicity of the DataEncryptionKeyLifecycle
// state machine (spec/contexts/data-keys.yaml): each mutating method
// performs its state transition (and, for Bootstrap/Rotate, the "at most one
// active version per tenant" invariant) as a single atomic operation, the
// same shape as backend/signingkeys/ports.KeyStore's Rotate.
type DataKeyRepository interface {
	// Bootstrap creates version 1 as active. Fails with
	// ErrDataKeyAlreadyBootstrapped if the tenant already has a
	// non-destroyed key.
	Bootstrap(ctx context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (*domain.TenantDataEncryptionKey, error)

	// Rotate creates a new active version and demotes the current active
	// version to retiring, atomically. Fails with ErrNoActiveDataKey if the
	// tenant has no active version.
	Rotate(ctx context.Context, tenantID string, wrappedDEK []byte, masterKeyID string, now time.Time) (next, previous *domain.TenantDataEncryptionKey, err error)

	// Disable locks out a retiring version immediately. Fails with
	// ErrDataKeyIsActive if the version is active, or
	// ErrDataKeyNotDisableable if it is any other non-retiring status.
	Disable(ctx context.Context, tenantID string, version int, now time.Time) (*domain.TenantDataEncryptionKey, error)

	// Destroy erases wrapped_dek for a retiring or disabled version
	// (crypto-shredding). Fails with ErrDataKeyIsActive if the version is
	// active, or ErrDataKeyNotDestroyable for any other ineligible status.
	Destroy(ctx context.Context, tenantID string, version int, now time.Time) (*domain.TenantDataEncryptionKey, error)

	// FindActive returns the tenant's active version, or
	// (nil, ErrNoActiveDataKey) if none exists.
	FindActive(ctx context.Context, tenantID string) (*domain.TenantDataEncryptionKey, error)

	// FindByVersion returns a specific version, or (nil, ErrDataKeyNotFound).
	FindByVersion(ctx context.Context, tenantID string, version int) (*domain.TenantDataEncryptionKey, error)

	// ListAll returns every version for a tenant, newest first.
	ListAll(ctx context.Context, tenantID string) ([]*domain.TenantDataEncryptionKey, error)
}
