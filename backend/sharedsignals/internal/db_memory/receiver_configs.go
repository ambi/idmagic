package db_memory

import (
	"context"
	"slices"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// SsfReceiverConfigRepository は SsfReceiverConfig の in-memory 実装。
type SsfReceiverConfigRepository struct {
	mu      sync.RWMutex
	configs map[string]*ssdomain.SsfReceiverConfig // key: sharedmem.TenantKey(tenant_id, stream_id)
}

func NewSsfReceiverConfigRepository() *SsfReceiverConfigRepository {
	return &SsfReceiverConfigRepository{configs: map[string]*ssdomain.SsfReceiverConfig{}}
}

func cloneReceiverConfig(c *ssdomain.SsfReceiverConfig) *ssdomain.SsfReceiverConfig {
	cloned := *c
	cloned.AcceptedAudiences = slices.Clone(c.AcceptedAudiences)
	return &cloned
}

func (r *SsfReceiverConfigRepository) FindByStream(_ context.Context, tenantID, streamID string) (*ssdomain.SsfReceiverConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c := r.configs[sharedmem.TenantKey(tenantID, streamID)]
	if c == nil {
		return nil, nil
	}
	return cloneReceiverConfig(c), nil
}

func (r *SsfReceiverConfigRepository) Save(_ context.Context, tenantID string, c *ssdomain.SsfReceiverConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[sharedmem.TenantKey(tenantID, c.StreamID)] = cloneReceiverConfig(c)
	return nil
}

func (r *SsfReceiverConfigRepository) Delete(_ context.Context, tenantID, streamID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.configs, sharedmem.TenantKey(tenantID, streamID))
	return nil
}
