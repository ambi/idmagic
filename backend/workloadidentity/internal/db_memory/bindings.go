package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
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
	b := r.bindings[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]
	if b == nil {
		return nil, nil
	}
	return cloneBinding(b), nil
}

func (r *AgentWorkloadBindingRepository) Save(_ context.Context, b *workloaddomain.AgentWorkloadBinding) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(b.TenantID, b.ID, kernel.DefaultTenantID)
	if current := r.bindings[key]; current != nil {
		updated := cloneBinding(current)
		updated.Status, updated.UpdatedAt, updated.DisabledAt = b.Status, b.UpdatedAt, b.DisabledAt
		r.bindings[key] = updated
		return nil
	}
	if r.patternTaken(b.TenantID, b.TrustBundleID, b.ID, b.SubjectPattern) {
		return ports.ErrBindingPatternConflict
	}
	r.bindings[key] = cloneBinding(b)
	return nil
}

func (r *AgentWorkloadBindingRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bindings, sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID))
	return nil
}

func (r *AgentWorkloadBindingRepository) UpdateMapping(_ context.Context, tenantID, id string, pattern, agentID *string, at time.Time) (*workloaddomain.AgentWorkloadBinding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)
	current := r.bindings[key]
	if current == nil {
		return nil, nil
	}
	if pattern != nil && r.patternTaken(tenantID, current.TrustBundleID, id, *pattern) {
		return nil, ports.ErrBindingPatternConflict
	}
	updated := cloneBinding(current)
	if pattern != nil {
		updated.SubjectPattern = *pattern
	}
	if agentID != nil {
		updated.AgentID = *agentID
	}
	updated.UpdatedAt = &at
	r.bindings[key] = updated
	return cloneBinding(updated), nil
}

// 呼び出し元でロックを保持し、作成と更新の一意性を同じ規則で守る。
func (r *AgentWorkloadBindingRepository) patternTaken(tenantID, bundleID, id, pattern string) bool {
	for _, other := range r.bindings {
		if other.TenantID == tenantID && other.TrustBundleID == bundleID && other.ID != id && other.SubjectPattern == pattern {
			return true
		}
	}
	return false
}
