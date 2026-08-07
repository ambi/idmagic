package db_memory

import (
	"context"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// AgentRevocationEpochRepository は AgentRevocationEpoch の in-memory 実装。
type AgentRevocationEpochRepository struct {
	mu     sync.RWMutex
	epochs map[string]*ssdomain.AgentRevocationEpoch // key: sharedmem.TenantKey(tenant_id, agent_id)
}

func NewAgentRevocationEpochRepository() *AgentRevocationEpochRepository {
	return &AgentRevocationEpochRepository{epochs: map[string]*ssdomain.AgentRevocationEpoch{}}
}

func cloneRevocationEpoch(e *ssdomain.AgentRevocationEpoch) *ssdomain.AgentRevocationEpoch {
	cloned := *e
	return &cloned
}

func (r *AgentRevocationEpochRepository) FindByAgent(_ context.Context, tenantID, agentID string) (*ssdomain.AgentRevocationEpoch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e := r.epochs[sharedmem.TenantKey(tenantID, agentID)]
	if e == nil {
		return nil, nil
	}
	return cloneRevocationEpoch(e), nil
}

// Advance は epoch を fail-closed に前進させる: 既存 epoch が存在し、新しい epoch が
// それ以降でなければ ErrEpochNotAdvancing を返し既存値を保持する (単調増加保証)。
func (r *AgentRevocationEpochRepository) Advance(_ context.Context, epoch ssdomain.AgentRevocationEpoch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.TenantKey(epoch.TenantID, epoch.AgentID)
	if existing := r.epochs[key]; existing != nil && epoch.Epoch.Before(existing.Epoch) {
		return ssdomain.ErrEpochNotAdvancing
	}
	r.epochs[key] = cloneRevocationEpoch(&epoch)
	return nil
}
