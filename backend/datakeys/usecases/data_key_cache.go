package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

type tenantVersion struct {
	tenantID string
	version  int
}

// DataKeyCache caches each (tenant, version) DEK it has unwrapped, so
// record-owning repositories are not forced through
// MasterKeyProvider.Unwrap on every encrypt/decrypt call. It implements
// ports.CacheInvalidator so lifecycle usecases can drop a tenant's entries
// on rotate/disable/destroy.
type DataKeyCache struct {
	mu            sync.RWMutex
	activeVersion map[string]int
	byVersion     map[tenantVersion][]byte
	repository    ports.DataKeyRepository
	crypto        envelope_crypto.EnvelopeCrypto
}

func NewDataKeyCache(repository ports.DataKeyRepository, crypto envelope_crypto.EnvelopeCrypto) *DataKeyCache {
	return &DataKeyCache{
		activeVersion: map[string]int{},
		byVersion:     map[tenantVersion][]byte{},
		repository:    repository,
		crypto:        crypto,
	}
}

// GetActive returns the tenant's active DEK version and plaintext, used to
// encrypt new ciphertext. It unwraps and caches on first use.
func (c *DataKeyCache) GetActive(ctx context.Context, tenantID string) (version int, plaintextDEK []byte, err error) {
	c.mu.RLock()
	if v, ok := c.activeVersion[tenantID]; ok {
		if dek, ok := c.byVersion[tenantVersion{tenantID, v}]; ok {
			c.mu.RUnlock()
			return v, dek, nil
		}
	}
	c.mu.RUnlock()

	key, err := c.repository.FindActive(ctx, tenantID)
	if err != nil {
		return 0, nil, err
	}
	plaintextDEK, err = c.crypto.Unwrap(ctx, tenantID, key.WrappedDEK, key.MasterKeyID)
	if err != nil {
		return 0, nil, err
	}

	c.mu.Lock()
	c.activeVersion[tenantID] = key.Version
	c.byVersion[tenantVersion{tenantID, key.Version}] = plaintextDEK
	c.mu.Unlock()
	return key.Version, plaintextDEK, nil
}

// GetByVersion は (テナント, 版) の平文 DEK を返す。回転して retiring になった版の暗号文の復号に使う。
// disabled（即時ロックアウト）と destroyed（crypto-shredding）の版は、wrapped_dek が残っていても
// フェイルクローズで拒否する。キャッシュはライフサイクルの操作が無効化するので、ここでは保存先の状態だけを見る。
func (c *DataKeyCache) GetByVersion(ctx context.Context, tenantID string, version int) ([]byte, error) {
	c.mu.RLock()
	if dek, ok := c.byVersion[tenantVersion{tenantID, version}]; ok {
		c.mu.RUnlock()
		return dek, nil
	}
	c.mu.RUnlock()

	key, err := c.repository.FindByVersion(ctx, tenantID, version)
	if err != nil {
		return nil, err
	}
	if key.Status == domain.DataKeyStatusDisabled || key.Status == domain.DataKeyStatusDestroyed {
		return nil, fmt.Errorf("%w: datakeys: version %d is %s", envelope_crypto.ErrDataKeyUnavailable, version, key.Status)
	}
	plaintextDEK, err := c.crypto.Unwrap(ctx, tenantID, key.WrappedDEK, key.MasterKeyID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.byVersion[tenantVersion{tenantID, version}] = plaintextDEK
	c.mu.Unlock()
	return plaintextDEK, nil
}

// Invalidate drops every cached entry for a tenant (all versions), forcing
// the next GetActive/GetByVersion to re-resolve from the repository.
func (c *DataKeyCache) Invalidate(tenantID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.activeVersion, tenantID)
	for key := range c.byVersion {
		if key.tenantID == tenantID {
			delete(c.byVersion, key)
		}
	}
}
