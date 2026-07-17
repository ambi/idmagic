package memory

import (
	"context"
	"slices"
	"sync"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// =====================================================================
// TenantUserAttributeSchemaRepository (ADR-040 / wi-19)
// =====================================================================

type TenantUserAttributeSchemaRepository struct {
	mu       sync.RWMutex
	byTenant map[string]*idmdomain.TenantUserAttributeSchema
}

func NewTenantUserAttributeSchemaRepository() *TenantUserAttributeSchemaRepository {
	return &TenantUserAttributeSchemaRepository{byTenant: map[string]*idmdomain.TenantUserAttributeSchema{}}
}

func (r *TenantUserAttributeSchemaRepository) FindByTenant(_ context.Context, tenantID string) (*idmdomain.TenantUserAttributeSchema, error) {
	sharedmem.DefaultTenant(&tenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if schema := r.byTenant[tenantID]; schema != nil {
		return cloneUserAttributeSchema(schema), nil
	}
	return nil, nil
}

func (r *TenantUserAttributeSchemaRepository) Save(_ context.Context, schema *idmdomain.TenantUserAttributeSchema) error {
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
func cloneUserAttributeSchema(s *idmdomain.TenantUserAttributeSchema) *idmdomain.TenantUserAttributeSchema {
	cloned := *s
	cloned.Attributes = slices.Clone(s.Attributes)
	return &cloned
}
