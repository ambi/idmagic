package datakeys

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/datakeys/usecases"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// FieldCipher is the public entry point record-owning repositories use to
// envelope-encrypt a single field. It composes the tenant DEK
// cache with the Tink-backed EnvelopeCrypto port and binds the fixed AAD
// (tenant/context/table/record id/field). Consuming contexts should depend
// on a small local interface (e.g.
// backend/authentication/totp/ports.SecretCipher) rather than this concrete
// type directly.
type FieldCipher struct {
	Repository ports.DataKeyRepository
	Cache      *usecases.DataKeyCache
	Crypto     envelope_crypto.EnvelopeCrypto
	// Emit optionally publishes DataEncryptionKeyBootstrapped when Encrypt
	// lazily provisions a tenant's first DEK.
	Emit func(spec.DomainEvent)
}

// Encrypt encrypts plaintext with the tenant's active DEK and returns the
// DEK version used (so the caller can persist it alongside the ciphertext
// for later decryption, even after the DEK rotates out to retiring). A
// tenant's first-ever call lazily bootstraps its DEK, mirroring SigningKeys'
// per-tenant key creation (docs/contexts/data-keys/internals.md) - no separate
// provisioning step is required.
func (f *FieldCipher) Encrypt(ctx context.Context, tenantID, recordContext, table, recordID, field, plaintext string) (keyVersion int, ciphertext []byte, err error) {
	version, dek, err := f.Cache.GetActive(ctx, tenantID)
	if errors.Is(err, domain.ErrNoActiveDataKey) {
		bootstrapDeps := usecases.Deps{Repository: f.Repository, Crypto: f.Crypto, Cache: f.Cache, Emit: f.Emit}
		_, bootstrapErr := usecases.BootstrapTenantDataKey(ctx, bootstrapDeps, tenantID, time.Now().UTC())
		// A concurrent request may have bootstrapped the tenant's DEK first;
		// that is not a failure here, just proceed to read the now-active key.
		if bootstrapErr != nil && !errors.Is(bootstrapErr, domain.ErrDataKeyAlreadyBootstrapped) {
			return 0, nil, bootstrapErr
		}
		version, dek, err = f.Cache.GetActive(ctx, tenantID)
	}
	if err != nil {
		return 0, nil, err
	}
	aad := envelope_crypto.AAD{TenantID: tenantID, Context: recordContext, Table: table, RecordID: recordID, Field: field}
	ciphertext, err = f.Crypto.Encrypt(ctx, dek, aad, []byte(plaintext))
	if err != nil {
		return 0, nil, err
	}
	return version, ciphertext, nil
}

// Decrypt decrypts ciphertext using the DEK version recorded alongside it,
// which may be retiring rather than the tenant's current active version.
func (f *FieldCipher) Decrypt(ctx context.Context, tenantID, recordContext, table, recordID, field string, keyVersion int, ciphertext []byte) (string, error) {
	dek, err := f.Cache.GetByVersion(ctx, tenantID, keyVersion)
	if err != nil {
		return "", err
	}
	aad := envelope_crypto.AAD{TenantID: tenantID, Context: recordContext, Table: table, RecordID: recordID, Field: field}
	plaintext, err := f.Crypto.Decrypt(ctx, dek, aad, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
