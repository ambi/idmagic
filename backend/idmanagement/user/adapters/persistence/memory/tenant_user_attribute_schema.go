package memory

import (
	"context"
	"slices"
	"sync"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// =====================================================================
// TenantUserAttributeSchemaRepository (ADR-040 / wi-19)
// =====================================================================

type TenantUserAttributeSchemaRepository struct {
	mu       sync.RWMutex
	byTenant map[string]*userdomain.TenantUserAttributeSchema
}

func NewTenantUserAttributeSchemaRepository() *TenantUserAttributeSchemaRepository {
	return &TenantUserAttributeSchemaRepository{byTenant: map[string]*userdomain.TenantUserAttributeSchema{}}
}

func (r *TenantUserAttributeSchemaRepository) FindByTenant(_ context.Context, tenantID string) (*userdomain.TenantUserAttributeSchema, error) {
	sharedmem.DefaultTenant(&tenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if schema := r.byTenant[tenantID]; schema != nil {
		return cloneUserAttributeSchema(schema), nil
	}
	return nil, nil
}

func (r *TenantUserAttributeSchemaRepository) Save(_ context.Context, schema *userdomain.TenantUserAttributeSchema) error {
	cloned := cloneUserAttributeSchema(schema)
	sharedmem.DefaultTenant(&cloned.TenantID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.byTenant[cloned.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.byTenant[cloned.TenantID] = cloned
	return nil
}

func (r *TenantUserAttributeSchemaRepository) Delete(_ context.Context, tenantID string) error {
	sharedmem.DefaultTenant(&tenantID)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byTenant, tenantID)
	return nil
}

// cloneUserAttributeSchema は呼び出し側との aliasing を断つための深いコピー。
func cloneUserAttributeSchema(s *userdomain.TenantUserAttributeSchema) *userdomain.TenantUserAttributeSchema {
	cloned := *s
	cloned.Attributes = slices.Clone(s.Attributes)
	return &cloned
}
