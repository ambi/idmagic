package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// =====================================================================
// TenantRepository (Tenancy)
// =====================================================================

type TenantRepository struct {
	mu      sync.RWMutex
	tenants map[string]*domain.Tenant
}

func NewTenantRepository() *TenantRepository {
	return &TenantRepository{tenants: map[string]*domain.Tenant{}}
}

func (r *TenantRepository) FindByID(_ context.Context, id string) (*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if tenant := r.tenants[id]; tenant != nil {
		cloned := *tenant
		return &cloned, nil
	}
	return nil, nil
}

func (r *TenantRepository) FindByRealm(_ context.Context, realm string) (*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, tenant := range r.tenants {
		if tenant.Realm == realm {
			cloned := *tenant
			return &cloned, nil
		}
	}
	return nil, nil
}

func (r *TenantRepository) FindAll(_ context.Context) ([]*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Tenant, 0, len(r.tenants))
	for _, tenant := range r.tenants {
		cloned := *tenant
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *domain.Tenant) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *TenantRepository) Save(_ context.Context, tenant *domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *tenant
	r.tenants[tenant.ID] = &cloned
	return nil
}
