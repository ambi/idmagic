package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

type WorkloadTrustBundleRepository struct {
	mu      sync.RWMutex
	bundles map[string]*workloaddomain.WorkloadTrustBundle // key: sharedmem.TenantKey(tenant_id, id)
}

func NewWorkloadTrustBundleRepository() *WorkloadTrustBundleRepository {
	return &WorkloadTrustBundleRepository{bundles: map[string]*workloaddomain.WorkloadTrustBundle{}}
}

func cloneTrustBundle(b *workloaddomain.WorkloadTrustBundle) *workloaddomain.WorkloadTrustBundle {
	cloned := *b
	cloned.AcceptedAudiences = slices.Clone(b.AcceptedAudiences)
	return &cloned
}

func (r *WorkloadTrustBundleRepository) ListAll(_ context.Context, tenantID string) ([]*workloaddomain.WorkloadTrustBundle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*workloaddomain.WorkloadTrustBundle, 0)
	for _, b := range r.bundles {
		if b.TenantID == tenantID {
			out = append(out, cloneTrustBundle(b))
		}
	}
	slices.SortFunc(out, func(a, b *workloaddomain.WorkloadTrustBundle) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (r *WorkloadTrustBundleRepository) FindByID(_ context.Context, tenantID, id string) (*workloaddomain.WorkloadTrustBundle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b := r.bundles[sharedmem.TenantKey(tenantID, id)]
	if b == nil {
		return nil, nil
	}
	return cloneTrustBundle(b), nil
}

func (r *WorkloadTrustBundleRepository) FindByIssuer(_ context.Context, tenantID, issuer string) (*workloaddomain.WorkloadTrustBundle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.bundles {
		if b.TenantID == tenantID && b.Issuer == issuer {
			return cloneTrustBundle(b), nil
		}
	}
	return nil, nil
}

func (r *WorkloadTrustBundleRepository) Save(_ context.Context, b *workloaddomain.WorkloadTrustBundle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bundles[sharedmem.TenantKey(b.TenantID, b.ID)] = cloneTrustBundle(b)
	return nil
}

func (r *WorkloadTrustBundleRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bundles, sharedmem.TenantKey(tenantID, id))
	return nil
}
