package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

type AgentWorkloadBindingRepository struct {
	mu       sync.RWMutex
	bindings map[string]*workloaddomain.AgentWorkloadBinding // key: sharedmem.TenantKey(tenant_id, id)
}

func NewAgentWorkloadBindingRepository() *AgentWorkloadBindingRepository {
	return &AgentWorkloadBindingRepository{bindings: map[string]*workloaddomain.AgentWorkloadBinding{}}
}

func cloneBinding(b *workloaddomain.AgentWorkloadBinding) *workloaddomain.AgentWorkloadBinding {
	cloned := *b
	return &cloned
}

func (r *AgentWorkloadBindingRepository) ListByTrustBundle(_ context.Context, tenantID, trustBundleID string) ([]*workloaddomain.AgentWorkloadBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*workloaddomain.AgentWorkloadBinding, 0)
	for _, b := range r.bindings {
		if b.TenantID == tenantID && b.TrustBundleID == trustBundleID {
			out = append(out, cloneBinding(b))
		}
	}
	slices.SortFunc(out, func(a, b *workloaddomain.AgentWorkloadBinding) int {
		return strings.Compare(a.SubjectPattern, b.SubjectPattern)
	})
	return out, nil
}

func (r *AgentWorkloadBindingRepository) FindByID(_ context.Context, tenantID, id string) (*workloaddomain.AgentWorkloadBinding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b := r.bindings[sharedmem.TenantKey(tenantID, id)]
	if b == nil {
		return nil, nil
	}
	return cloneBinding(b), nil
}

func (r *AgentWorkloadBindingRepository) Save(_ context.Context, b *workloaddomain.AgentWorkloadBinding) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bindings[sharedmem.TenantKey(b.TenantID, b.ID)] = cloneBinding(b)
	return nil
}

func (r *AgentWorkloadBindingRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bindings, sharedmem.TenantKey(tenantID, id))
	return nil
}
