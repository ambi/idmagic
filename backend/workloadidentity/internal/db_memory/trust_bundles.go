package db_memory

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

var (
	errDuplicateTrustBundleIssuer      = errors.New("workload trust bundle issuer already registered in this tenant")
	errDuplicateTrustBundleTrustDomain = errors.New("spiffe workload trust bundle trust domain already registered in this tenant")
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
	cloned.JWKSCached = slices.Clone(b.JWKSCached)
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
		if b.TenantID == tenantID && issuer != "" && b.Issuer == issuer {
			return cloneTrustBundle(b), nil
		}
	}
	return nil, nil
}

func (r *WorkloadTrustBundleRepository) FindSPIFFEByTrustDomain(_ context.Context, tenantID, trustDomain string) (*workloaddomain.WorkloadTrustBundle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.bundles {
		if b.TenantID == tenantID && b.IsSPIFFE() && b.TrustDomain == trustDomain {
			return cloneTrustBundle(b), nil
		}
	}
	return nil, nil
}

func (r *WorkloadTrustBundleRepository) Save(_ context.Context, b *workloaddomain.WorkloadTrustBundle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, stored := range r.bundles {
		if stored.TenantID != b.TenantID || stored.ID == b.ID {
			continue
		}
		if b.Issuer != "" && stored.Issuer == b.Issuer {
			return errDuplicateTrustBundleIssuer
		}
		if b.IsSPIFFE() && stored.IsSPIFFE() && stored.TrustDomain == b.TrustDomain {
			return errDuplicateTrustBundleTrustDomain
		}
	}
	key := sharedmem.ScopedKey(b.TenantID, b.ID, kernel.DefaultTenantID)
	saved := cloneTrustBundle(b)
	// 更新では、最後に取得できた鍵を読んだ値で上書きせず、jwks_uri が変わるときだけ消す
	// (PostgreSQL の SaveWorkloadTrustBundle と同じ)。
	if existing := r.bundles[key]; existing != nil {
		saved.JWKSCached, saved.JWKSCachedAt = existing.JWKSCached, existing.JWKSCachedAt
		if saved.JWKSSourceChanged(*existing) {
			saved.JWKSCached, saved.JWKSCachedAt = nil, nil
		}
	}
	r.bundles[key] = saved
	return nil
}

func (r *WorkloadTrustBundleRepository) RecordJWKSFetched(_ context.Context, tenantID, id string, jwksURI *string, fetched workloaddomain.JWKSSnapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.bundles[sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID)]
	if b == nil || b.JWKSSourceChanged(workloaddomain.WorkloadTrustBundle{JWKSURI: jwksURI}) {
		return nil
	}
	if b.JWKSCachedAt != nil && b.JWKSCachedAt.After(fetched.FetchedAt) {
		return nil
	}
	keys := fetched.Keys
	if keys == nil {
		keys = []map[string]any{}
	}
	at := fetched.FetchedAt
	b.JWKSCached, b.JWKSCachedAt = slices.Clone(keys), &at
	return nil
}

func (r *WorkloadTrustBundleRepository) Delete(_ context.Context, tenantID, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.bundles, sharedmem.ScopedKey(tenantID, id, kernel.DefaultTenantID))
	return nil
}
