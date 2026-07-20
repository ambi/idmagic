package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/saml/domain"
	sharedmemory "github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

// =====================================================================
// SamlServiceProviderRepository (SAML 2.0 Web Browser SSO, wi-29)
// =====================================================================

type SamlServiceProviderRepository struct {
	mu  sync.RWMutex
	sps map[string]*domain.SamlServiceProvider // key: TenantKey(tenant_id, entity_id)
}

func NewSamlServiceProviderRepository() *SamlServiceProviderRepository {
	return &SamlServiceProviderRepository{sps: map[string]*domain.SamlServiceProvider{}}
}

// Seed は起動時のサンプル SP 投入に使う (テスト・デモ用)。
func (r *SamlServiceProviderRepository) Seed(sp *domain.SamlServiceProvider) {
	_ = r.Save(context.Background(), sp)
}

func cloneServiceProvider(sp *domain.SamlServiceProvider) *domain.SamlServiceProvider {
	cloned := *sp
	cloned.ACSURLs = slices.Clone(sp.ACSURLs)
	cloned.ClaimPolicy.Rules = slices.Clone(sp.ClaimPolicy.Rules)
	return &cloned
}

func (r *SamlServiceProviderRepository) FindByEntityID(_ context.Context, tenantID, entityID string) (*domain.SamlServiceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sp := r.sps[sharedmemory.TenantKey(tenantID, entityID)]
	if sp == nil {
		return nil, nil
	}
	return cloneServiceProvider(sp), nil
}

func (r *SamlServiceProviderRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.SamlServiceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.SamlServiceProvider, 0)
	for _, sp := range r.sps {
		if sp.TenantID == tenantID {
			out = append(out, cloneServiceProvider(sp))
		}
	}
	slices.SortFunc(out, func(a, b *domain.SamlServiceProvider) int { return strings.Compare(a.EntityID, b.EntityID) })
	return out, nil
}

func (r *SamlServiceProviderRepository) Save(_ context.Context, sp *domain.SamlServiceProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sharedmemory.DefaultTenant(&sp.TenantID)
	r.sps[sharedmemory.TenantKey(sp.TenantID, sp.EntityID)] = cloneServiceProvider(sp)
	return nil
}

func (r *SamlServiceProviderRepository) Delete(_ context.Context, tenantID, entityID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sps, sharedmemory.TenantKey(tenantID, entityID))
	return nil
}
