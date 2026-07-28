package usecases

import (
	"context"
	"sync"

	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

type cachedDataKey struct {
	keyID        string
	plaintextDEK []byte
}

// DataKeyCache caches each tenant's unwrapped active DEK so record-owning
// repositories are not forced through MasterKeyProvider.Unwrap on every
// encrypt/decrypt call. It implements ports.CacheInvalidator so lifecycle
// usecases can drop an entry on rotate/disable/destroy (ADR-148).
type DataKeyCache struct {
	mu         sync.RWMutex
	active     map[string]cachedDataKey
	repository ports.DataKeyRepository
	crypto     envelope_crypto.EnvelopeCrypto
}

func NewDataKeyCache(repository ports.DataKeyRepository, crypto envelope_crypto.EnvelopeCrypto) *DataKeyCache {
	return &DataKeyCache{
		active:     map[string]cachedDataKey{},
		repository: repository,
		crypto:     crypto,
	}
}

// GetActive returns the tenant's active DEK id and plaintext, unwrapping and
// caching it on first use. A fail-closed error (unwrap failure, no active
// key) is never cached.
func (c *DataKeyCache) GetActive(ctx context.Context, tenantID string) (keyID string, plaintextDEK []byte, err error) {
	c.mu.RLock()
	if cached, ok := c.active[tenantID]; ok {
		c.mu.RUnlock()
		return cached.keyID, cached.plaintextDEK, nil
	}
	c.mu.RUnlock()

	key, err := c.repository.FindActive(ctx, tenantID)
	if err != nil {
		return "", nil, err
	}
	plaintextDEK, err = c.crypto.Unwrap(ctx, tenantID, key.WrappedDEK, key.MasterKeyID)
	if err != nil {
		return "", nil, err
	}

	c.mu.Lock()
	c.active[tenantID] = cachedDataKey{keyID: key.ID, plaintextDEK: plaintextDEK}
	c.mu.Unlock()
	return key.ID, plaintextDEK, nil
}

func (c *DataKeyCache) Invalidate(tenantID string) {
	c.mu.Lock()
	delete(c.active, tenantID)
	c.mu.Unlock()
}
