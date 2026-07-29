package envelope_openbao

import (
	"context"
	"fmt"

	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

// TransitEngine is the OpenBao wire operations OpenBaoMasterKeyProvider
// needs. It is an interface (rather than a direct *HTTPTransitEngine
// dependency) so tests can substitute a fake without a network call, mirroring
// backend/signingkeys/keys_vault's TransitEngine seam.
type TransitEngine interface {
	EnsureKey(ctx context.Context, name string) error
	EncryptDataKey(ctx context.Context, name string, plaintext []byte) (ciphertext string, err error)
	DecryptDataKey(ctx context.Context, name, ciphertext string) (plaintext []byte, err error)
	Healthy(ctx context.Context) bool
}

// OpenBaoMasterKeyProvider implements envelope_crypto.MasterKeyProvider using
// one OpenBao transit key per tenant, named "{keyNamePrefix}/{tenantID}".
type OpenBaoMasterKeyProvider struct {
	engine        TransitEngine
	keyNamePrefix string
}

func NewOpenBaoMasterKeyProvider(engine TransitEngine, keyNamePrefix string) *OpenBaoMasterKeyProvider {
	return &OpenBaoMasterKeyProvider{engine: engine, keyNamePrefix: keyNamePrefix}
}

func (p *OpenBaoMasterKeyProvider) WrapDataKey(ctx context.Context, tenantID string, plaintextDEK []byte) ([]byte, string, error) {
	name := p.keyName(tenantID)
	if err := p.engine.EnsureKey(ctx, name); err != nil {
		return nil, "", fmt.Errorf("envelope_openbao: ensure transit key: %w", err)
	}
	ciphertext, err := p.engine.EncryptDataKey(ctx, name, plaintextDEK)
	if err != nil {
		return nil, "", fmt.Errorf("envelope_openbao: encrypt data key: %w", err)
	}
	return []byte(ciphertext), name, nil
}

func (p *OpenBaoMasterKeyProvider) UnwrapDataKey(ctx context.Context, tenantID string, wrapped []byte, masterKeyID string) ([]byte, error) {
	name := p.keyName(tenantID)
	if masterKeyID != name {
		return nil, fmt.Errorf("%w: envelope_openbao: master key id %q does not match tenant %q", envelope_crypto.ErrDecryptionFailed, masterKeyID, tenantID)
	}
	plaintext, err := p.engine.DecryptDataKey(ctx, name, string(wrapped))
	if err != nil {
		return nil, fmt.Errorf("%w: envelope_openbao: decrypt data key: %w", envelope_crypto.ErrDecryptionFailed, err)
	}
	return plaintext, nil
}

func (p *OpenBaoMasterKeyProvider) Healthy(ctx context.Context) bool {
	return p.engine.Healthy(ctx)
}

func (p *OpenBaoMasterKeyProvider) Provider() string {
	return "openbao"
}

func (p *OpenBaoMasterKeyProvider) keyName(tenantID string) string {
	return p.keyNamePrefix + "/" + tenantID
}
