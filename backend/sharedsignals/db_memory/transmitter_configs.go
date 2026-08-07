package db_memory

import (
	"context"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// SsfTransmitterConfigRepository は SsfTransmitterConfig の in-memory 実装。
type SsfTransmitterConfigRepository struct {
	mu      sync.RWMutex
	configs map[string]*ssdomain.SsfTransmitterConfig // key: sharedmem.TenantKey(tenant_id, stream_id)
}

func NewSsfTransmitterConfigRepository() *SsfTransmitterConfigRepository {
	return &SsfTransmitterConfigRepository{configs: map[string]*ssdomain.SsfTransmitterConfig{}}
}

func (r *SsfTransmitterConfigRepository) FindByStream(_ context.Context, tenantID, streamID string) (*ssdomain.SsfTransmitterConfig, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c := r.configs[sharedmem.TenantKey(tenantID, streamID)]
	if c == nil {
		return nil, nil
	}
	cloned := *c
	return &cloned, nil
}

func (r *SsfTransmitterConfigRepository) Save(_ context.Context, tenantID string, c *ssdomain.SsfTransmitterConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *c
	r.configs[sharedmem.TenantKey(tenantID, c.StreamID)] = &cloned
	return nil
}

func (r *SsfTransmitterConfigRepository) Delete(_ context.Context, tenantID, streamID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.configs, sharedmem.TenantKey(tenantID, streamID))
	return nil
}
