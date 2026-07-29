// Package envelope_crypto: Layer 4 - Adapter Layer (technical shared adapter)
//
// EnvelopeCrypto is the port for two-tier envelope encryption of DB-resident
// reversible secrets (ADR-148): a MasterKeyProvider wraps per-tenant
// DataEncryptionKeys, and this package uses Tink's AEAD primitive directly to
// generate DEKs and encrypt/decrypt records. AEAD/keyset handling — nonce
// generation, authentication tags — is delegated entirely to Tink; this
// package never assembles those by hand.
package envelope_crypto

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/insecurecleartextkeyset"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"
)

// ErrDecryptionFailed signals a fail-closed refusal (ADR-148): unwrap
// failure, AAD/tamper mismatch, or any other reason a secret cannot be
// recovered. Callers must deny access, never fall back to plaintext.
var ErrDecryptionFailed = errors.New("envelope_crypto: decryption failed (fail-closed)")

// MasterKeyProvider is the swappable custody boundary for the key material
// that wraps per-tenant DataEncryptionKeys (ADR-148). Implementations:
// envelope_openbao (production, Vault Transit-compatible) and
// envelope_cleartext (dev/local, no external service required).
type MasterKeyProvider interface {
	WrapDataKey(ctx context.Context, tenantID string, plaintextDEK []byte) (wrapped []byte, masterKeyID string, err error)
	UnwrapDataKey(ctx context.Context, tenantID string, wrapped []byte, masterKeyID string) (plaintextDEK []byte, err error)
	Healthy(ctx context.Context) bool
	// Provider names this MasterKeyProvider for health/observability display
	// (spec/contexts/data-keys.yaml TenantDataKeyHealth.provider, wi-97 T007)
	// — e.g. "openbao" or "tink_cleartext". Never reveals key material.
	Provider() string
}

// EnvelopeCrypto is the port DataKeys and record-owning repositories depend
// on (ADR-148). GenerateDataKey/Encrypt/Decrypt are always Tink-backed;
// Wrap/Unwrap/Healthy delegate to the injected MasterKeyProvider.
type EnvelopeCrypto interface {
	GenerateDataKey(ctx context.Context) (plaintextDEK []byte, err error)
	Wrap(ctx context.Context, tenantID string, plaintextDEK []byte) (wrapped []byte, masterKeyID string, err error)
	Unwrap(ctx context.Context, tenantID string, wrapped []byte, masterKeyID string) (plaintextDEK []byte, err error)
	Encrypt(ctx context.Context, plaintextDEK []byte, aad AAD, plaintext []byte) (ciphertext []byte, err error)
	Decrypt(ctx context.Context, plaintextDEK []byte, aad AAD, ciphertext []byte) (plaintext []byte, err error)
	Healthy(ctx context.Context) bool
	Provider() string
}

// AAD is the fixed associated-data binding for record-level ciphertext
// (ADR-148): tenant, owning context, table, record id, and field. Binding all
// five means a ciphertext copied across any one of these dimensions fails to
// decrypt instead of silently succeeding.
type AAD struct {
	TenantID string
	Context  string
	Table    string
	RecordID string
	Field    string
}

func (a AAD) bytes() []byte {
	return fmt.Appendf(nil, "idmagic:datakeys:v1:%s:%s:%s:%s:%s", a.TenantID, a.Context, a.Table, a.RecordID, a.Field)
}

// TinkEnvelopeCrypto is the EnvelopeCrypto implementation. DEK generation and
// record encryption always use Tink's AES256-GCM AEAD primitive; master-key
// custody is delegated to the injected MasterKeyProvider.
type TinkEnvelopeCrypto struct {
	masterKey MasterKeyProvider
}

func NewTinkEnvelopeCrypto(masterKey MasterKeyProvider) *TinkEnvelopeCrypto {
	return &TinkEnvelopeCrypto{masterKey: masterKey}
}

// GenerateDataKey returns a new plaintext DEK serialized as a Tink cleartext
// keyset. Callers must Wrap it before persisting; the plaintext form must
// never reach storage, logs, or events.
func (c *TinkEnvelopeCrypto) GenerateDataKey(_ context.Context) ([]byte, error) {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return nil, fmt.Errorf("envelope_crypto: generate data key: %w", err)
	}
	buf := &bytes.Buffer{}
	if err := insecurecleartextkeyset.Write(handle, keyset.NewBinaryWriter(buf)); err != nil {
		return nil, fmt.Errorf("envelope_crypto: serialize data key: %w", err)
	}
	return buf.Bytes(), nil
}

func (c *TinkEnvelopeCrypto) Wrap(ctx context.Context, tenantID string, plaintextDEK []byte) ([]byte, string, error) {
	wrapped, masterKeyID, err := c.masterKey.WrapDataKey(ctx, tenantID, plaintextDEK)
	if err != nil {
		return nil, "", fmt.Errorf("envelope_crypto: wrap data key: %w", err)
	}
	return wrapped, masterKeyID, nil
}

func (c *TinkEnvelopeCrypto) Unwrap(ctx context.Context, tenantID string, wrapped []byte, masterKeyID string) ([]byte, error) {
	plaintextDEK, err := c.masterKey.UnwrapDataKey(ctx, tenantID, wrapped, masterKeyID)
	if err != nil {
		return nil, fmt.Errorf("%w: unwrap data key: %w", ErrDecryptionFailed, err)
	}
	return plaintextDEK, nil
}

func (c *TinkEnvelopeCrypto) Encrypt(_ context.Context, plaintextDEK []byte, aad AAD, plaintext []byte) ([]byte, error) {
	primitive, err := aeadPrimitive(plaintextDEK)
	if err != nil {
		return nil, err
	}
	ciphertext, err := primitive.Encrypt(plaintext, aad.bytes())
	if err != nil {
		return nil, fmt.Errorf("envelope_crypto: encrypt: %w", err)
	}
	return ciphertext, nil
}

func (c *TinkEnvelopeCrypto) Decrypt(_ context.Context, plaintextDEK []byte, aad AAD, ciphertext []byte) ([]byte, error) {
	primitive, err := aeadPrimitive(plaintextDEK)
	if err != nil {
		return nil, err
	}
	plaintext, err := primitive.Decrypt(ciphertext, aad.bytes())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

func (c *TinkEnvelopeCrypto) Healthy(ctx context.Context) bool {
	return c.masterKey.Healthy(ctx)
}

func (c *TinkEnvelopeCrypto) Provider() string {
	return c.masterKey.Provider()
}

func aeadPrimitive(serializedKeyset []byte) (tink.AEAD, error) {
	handle, err := insecurecleartextkeyset.Read(keyset.NewBinaryReader(bytes.NewReader(serializedKeyset)))
	if err != nil {
		return nil, fmt.Errorf("envelope_crypto: read data key: %w", err)
	}
	primitive, err := aead.New(handle)
	if err != nil {
		return nil, fmt.Errorf("envelope_crypto: build aead primitive: %w", err)
	}
	return primitive, nil
}
