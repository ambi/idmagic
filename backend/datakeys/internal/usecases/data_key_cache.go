package usecases

import (
	"context"
	"fmt"
	"sync"

	"github.com/ambi/idmagic/backend/datakeys/internal/domain"
	"github.com/ambi/idmagic/backend/datakeys/internal/ports"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
)

type tenantVersion struct {
	tenantID string
	version  int
}

// DataKeyCache は、アンラップした DEK を (テナント, 版) ごとに保持し、暗号化と復号のたびに
// MasterKeyProvider を呼ばずに済ませる。版の鍵素材は変わらないが状態は変わるので、状態は保持せず、
// 呼び出しのたびに保存先から読む。ほかのプロセスのライフサイクルの操作は、これで確定の直後から効く。
// ports.CacheInvalidator の実装は、操作したプロセスの鍵素材を早く捨てるためにある。
type DataKeyCache struct {
	mu         sync.RWMutex
	byVersion  map[tenantVersion][]byte
	repository ports.DataKeyRepository
	crypto     envelope_crypto.EnvelopeCrypto
}

func NewDataKeyCache(repository ports.DataKeyRepository, crypto envelope_crypto.EnvelopeCrypto) *DataKeyCache {
	return &DataKeyCache{
		byVersion:  map[tenantVersion][]byte{},
		repository: repository,
		crypto:     crypto,
	}
}

// GetActive returns the tenant's active DEK version and plaintext, used to
// encrypt new ciphertext. The active version always comes from the repository.
func (c *DataKeyCache) GetActive(ctx context.Context, tenantID string) (version int, plaintextDEK []byte, err error) {
	key, err := c.repository.FindActive(ctx, tenantID)
	if err != nil {
		return 0, nil, err
	}
	plaintextDEK, err = c.plaintext(ctx, key)
	if err != nil {
		return 0, nil, err
	}
	return key.Version, plaintextDEK, nil
}

// GetByVersion は (テナント, 版) の平文 DEK を返す。回転して retiring になった版の暗号文の復号に使う。
// disabled（即時ロックアウト）と destroyed（crypto-shredding）の版は、DEK を保持していても
// フェイルクローズで拒否し、保持も捨てる。
func (c *DataKeyCache) GetByVersion(ctx context.Context, tenantID string, version int) ([]byte, error) {
	key, err := c.repository.FindByVersion(ctx, tenantID, version)
	if err != nil {
		return nil, err
	}
	if key.Status == domain.DataKeyStatusDisabled || key.Status == domain.DataKeyStatusDestroyed {
		c.mu.Lock()
		delete(c.byVersion, tenantVersion{tenantID, version})
		c.mu.Unlock()
		return nil, fmt.Errorf("%w: datakeys: version %d is %s", envelope_crypto.ErrDataKeyUnavailable, version, key.Status)
	}
	return c.plaintext(ctx, key)
}

// plaintext は、保存先から読んだ版の平文 DEK を、保持になければアンラップして返す。
func (c *DataKeyCache) plaintext(ctx context.Context, key *domain.TenantDataEncryptionKey) ([]byte, error) {
	cacheKey := tenantVersion{key.TenantID, key.Version}
	c.mu.RLock()
	plaintextDEK, ok := c.byVersion[cacheKey]
	c.mu.RUnlock()
	if ok {
		return plaintextDEK, nil
	}

	plaintextDEK, err := c.crypto.Unwrap(ctx, key.TenantID, key.WrappedDEK, key.MasterKeyID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.byVersion[cacheKey] = plaintextDEK
	c.mu.Unlock()
	return plaintextDEK, nil
}

// Invalidate drops every cached DEK for a tenant (all versions).
func (c *DataKeyCache) Invalidate(tenantID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.byVersion {
		if key.tenantID == tenantID {
			delete(c.byVersion, key)
		}
	}
}
