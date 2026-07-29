// Package domain: Layer 3 - Domain Layer
//
// TenantDataEncryptionKey is the per-tenant DEK metadata that the DataKeys
// context owns (ADR-148, spec/contexts/data-keys.yaml). The plaintext DEK
// itself is never part of this struct; wrapped_dek only ever holds the
// master-key-wrapped form produced by envelope_crypto.EnvelopeCrypto.Wrap.
package domain

import (
	"errors"
	"time"
)

type DataKeyStatus string

const (
	DataKeyStatusActive    DataKeyStatus = "active"
	DataKeyStatusRetiring  DataKeyStatus = "retiring"
	DataKeyStatusDisabled  DataKeyStatus = "disabled"
	DataKeyStatusDestroyed DataKeyStatus = "destroyed"
)

var (
	// ErrDataKeyAlreadyBootstrapped: the tenant already has a non-destroyed
	// data key; BootstrapTenantDataKey is not idempotent re-creation.
	ErrDataKeyAlreadyBootstrapped = errors.New("datakeys: tenant already has a data key")
	// ErrNoActiveDataKey: RotateTenantDataKey requires an existing active key
	// to demote to retiring.
	ErrNoActiveDataKey = errors.New("datakeys: no active data key for tenant")
	// ErrDataKeyNotFound: no data key exists for the given tenant/version.
	ErrDataKeyNotFound = errors.New("datakeys: data key version not found")
	// ErrDataKeyIsActive: DisableTenantDataKey/DestroyTenantDataKey were
	// called on the tenant's active version; rotate it out first
	// (spec/contexts/data-keys.yaml DisableTenantDataKey/DestroyTenantDataKey
	// requires clause).
	ErrDataKeyIsActive = errors.New("datakeys: cannot disable or destroy an active data key")
	// ErrDataKeyNotDisableable: DisableTenantDataKey requires status=retiring.
	ErrDataKeyNotDisableable = errors.New("datakeys: data key must be retiring to disable")
	// ErrDataKeyNotDestroyable: DestroyTenantDataKey requires
	// status in {retiring, disabled}.
	ErrDataKeyNotDestroyable = errors.New("datakeys: data key must be retiring or disabled to destroy")
	// ErrDataKeyStillReferenced: DestroyTenantDataKey found a registered
	// FieldMigrator reporting rows not yet re-encrypted onto the tenant's
	// active version; destroying now would make those rows permanently
	// undecryptable (spec/contexts/data-keys.yaml DataKeyStillReferencedError,
	// wi-97 T006). Run the reencryption job to completion first.
	ErrDataKeyStillReferenced = errors.New("datakeys: cannot destroy data key while secrets are still pending re-encryption onto the active version")
)

// TenantDataEncryptionKey is one version of a tenant's DEK lifecycle
// (spec/contexts/data-keys.yaml TenantDataEncryptionKey).
type TenantDataEncryptionKey struct {
	ID          string
	TenantID    string
	Version     int
	Status      DataKeyStatus
	WrappedDEK  []byte
	MasterKeyID string
	CreatedAt   time.Time
	ActivatedAt *time.Time
	DisabledAt  *time.Time
	DestroyedAt *time.Time
}

// TenantDataKeyHealth is the system_admin-facing health snapshot
// (spec/contexts/data-keys.yaml TenantDataKeyHealth) — never carries key
// material.
type TenantDataKeyHealth struct {
	TenantID          string
	ActiveVersion     int
	Status            DataKeyStatus
	Provider          string
	ProviderReachable bool
	RotatedAt         *time.Time
}
