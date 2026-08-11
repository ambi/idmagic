// Package envelope_cleartext: Layer 4 - Adapter Layer (technical shared adapter)
//
// CleartextMasterKeyProvider is the dev/local envelope_crypto.MasterKeyProvider
// : it wraps per-tenant DEKs with an in-process Tink keyset instead
// of an external KMS, so engineers can develop without standing up OpenBao.
// It must never be selected in production configuration.
package envelope_cleartext

import (
	"context"
	"fmt"

	"github.com/tink-crypto/tink-go/v2/aead"
	"github.com/tink-crypto/tink-go/v2/keyset"
	"github.com/tink-crypto/tink-go/v2/tink"

	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

const masterKeyID = "cleartext-dev"

type CleartextMasterKeyProvider struct {
	primitive tink.AEAD
}

func NewCleartextMasterKeyProvider() (*CleartextMasterKeyProvider, error) {
	handle, err := keyset.NewHandle(aead.AES256GCMKeyTemplate())
	if err != nil {
		return nil, fmt.Errorf("envelope_cleartext: generate master key: %w", err)
	}
	primitive, err := aead.New(handle)
	if err != nil {
		return nil, fmt.Errorf("envelope_cleartext: build aead primitive: %w", err)
	}
	return &CleartextMasterKeyProvider{primitive: primitive}, nil
}

func (p *CleartextMasterKeyProvider) WrapDataKey(_ context.Context, tenantID string, plaintextDEK []byte) ([]byte, string, error) {
	wrapped, err := p.primitive.Encrypt(plaintextDEK, wrapAAD(tenantID))
	if err != nil {
		return nil, "", fmt.Errorf("envelope_cleartext: wrap data key: %w", err)
	}
	return wrapped, masterKeyID, nil
}

func (p *CleartextMasterKeyProvider) UnwrapDataKey(_ context.Context, tenantID string, wrapped []byte, gotMasterKeyID string) ([]byte, error) {
	if gotMasterKeyID != masterKeyID {
		return nil, fmt.Errorf("%w: envelope_cleartext: unknown master key id %q", envelope_crypto.ErrDecryptionFailed, gotMasterKeyID)
	}
	plaintext, err := p.primitive.Decrypt(wrapped, wrapAAD(tenantID))
	if err != nil {
		return nil, fmt.Errorf("%w: envelope_cleartext: unwrap data key: %w", envelope_crypto.ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

func (p *CleartextMasterKeyProvider) Healthy(_ context.Context) bool {
	return true
}

func (p *CleartextMasterKeyProvider) Provider() string {
	return "tink_cleartext"
}

func wrapAAD(tenantID string) []byte {
	return fmt.Appendf(nil, "idmagic:datakeys:master:cleartext:%s", tenantID)
}
