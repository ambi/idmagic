package db_memory

import (
	"context"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
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
	e := r.epochs[sharedmem.ScopedKey(tenantID, agentID, kernel.DefaultTenantID)]
	if e == nil {
		return nil, nil
	}
	return cloneRevocationEpoch(e), nil
}

// Advance は epoch を fail-closed に前進させる: 既存 epoch が存在し、新しい epoch が既存より
// 後でなければ ErrEpochNotAdvancing を返し既存値を保持する (単調増加保証)。
func (r *AgentRevocationEpochRepository) Advance(_ context.Context, epoch ssdomain.AgentRevocationEpoch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(epoch.TenantID, epoch.AgentID, kernel.DefaultTenantID)
	if existing := r.epochs[key]; existing != nil && !epoch.Epoch.After(existing.Epoch) {
		return ssdomain.ErrEpochNotAdvancing
	}
	r.epochs[key] = cloneRevocationEpoch(&epoch)
	return nil
}

// UserRevocationEpochRepository は UserRevocationEpoch の in-memory 実装。
type UserRevocationEpochRepository struct {
	mu     sync.RWMutex
	epochs map[string]ssdomain.UserRevocationEpoch // key: sharedmem.ScopedKey(tenant_id, user_id)
}

func NewUserRevocationEpochRepository() *UserRevocationEpochRepository {
	return &UserRevocationEpochRepository{epochs: map[string]ssdomain.UserRevocationEpoch{}}
}

func (r *UserRevocationEpochRepository) FindByUser(_ context.Context, tenantID, userID string) (*ssdomain.UserRevocationEpoch, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.epochs[sharedmem.ScopedKey(tenantID, userID, kernel.DefaultTenantID)]
	if !ok {
		return nil, nil
	}
	return &e, nil
}

// Advance は既存より後の epoch だけを受け付け、それ以外は ErrEpochNotAdvancing を返す。
func (r *UserRevocationEpochRepository) Advance(_ context.Context, epoch ssdomain.UserRevocationEpoch) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(epoch.TenantID, epoch.UserID, kernel.DefaultTenantID)
	if existing, ok := r.epochs[key]; ok && !epoch.Epoch.After(existing.Epoch) {
		return ssdomain.ErrEpochNotAdvancing
	}
	r.epochs[key] = epoch
	return nil
}
