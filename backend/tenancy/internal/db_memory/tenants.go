package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// =====================================================================
// TenantRepository (Tenancy)
// =====================================================================

type TenantRepository struct {
	mu      sync.RWMutex
	tenants map[string]*tenancydomain.Tenant
}

func NewTenantRepository() *TenantRepository {
	return &TenantRepository{tenants: map[string]*tenancydomain.Tenant{}}
}

func (r *TenantRepository) FindByID(_ context.Context, id string) (*tenancydomain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if tenant := r.tenants[id]; tenant != nil {
		cloned := *tenant
		return &cloned, nil
	}
	return nil, nil
}

func (r *TenantRepository) FindByRealm(_ context.Context, realm string) (*tenancydomain.Tenant, error) {
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

func (r *TenantRepository) FindAll(_ context.Context) ([]*tenancydomain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*tenancydomain.Tenant, 0, len(r.tenants))
	for _, tenant := range r.tenants {
		cloned := *tenant
		out = append(out, &cloned)
	}
	slices.SortFunc(out, func(a, b *tenancydomain.Tenant) int { return strings.Compare(a.ID, b.ID) })
	return out, nil
}

func (r *TenantRepository) Save(_ context.Context, tenant *tenancydomain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *tenant
	r.tenants[tenant.ID] = &cloned
	return nil
}

func (r *TenantRepository) Update(_ context.Context, id string, change func(*tenancydomain.Tenant) (*tenancydomain.Tenant, error)) (*tenancydomain.Tenant, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var current *tenancydomain.Tenant
	if stored := r.tenants[id]; stored != nil {
		cloned := *stored
		current = &cloned
	}
	updated, err := change(current)
	if err != nil {
		return nil, err
	}
	cloned := *updated
	r.tenants[id] = &cloned
	return updated, nil
}
