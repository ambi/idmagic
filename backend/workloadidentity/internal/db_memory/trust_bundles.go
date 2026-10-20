package db_memory

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

var errDuplicateTrustBundleIssuer = errors.New("workload trust bundle issuer already registered in this tenant")

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
	b := r.bundles[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]
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
	for _, stored := range r.bundles {
		if stored.TenantID == b.TenantID && stored.Issuer == b.Issuer && stored.ID != b.ID {
			return errDuplicateTrustBundleIssuer
		}
	}
	r.bundles[sharedmem.ScopedKey(b.TenantID, b.ID, kernel.DefaultTenantID)] = cloneTrustBundle(b)
	return nil
}

func (r *WorkloadTrustBundleRepository) RecordJWKSFetched(_ context.Context, tenantID, id string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b := r.bundles[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]; b != nil {
		b.JWKSCachedAt = &at
	}
	return nil
}

func (r *WorkloadTrustBundleRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bundles, sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID))
	return nil
}
