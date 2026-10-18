package db_memory

import (
	"context"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

// =====================================================================
// TenantGroupAttributeSchemaRepository (wi-315)
// =====================================================================

type TenantGroupAttributeSchemaRepository struct {
	mu       sync.RWMutex
	byTenant map[string]*groupdomain.TenantGroupAttributeSchema
}

func NewTenantGroupAttributeSchemaRepository() *TenantGroupAttributeSchemaRepository {
	return &TenantGroupAttributeSchemaRepository{byTenant: map[string]*groupdomain.TenantGroupAttributeSchema{}}
}

func (r *TenantGroupAttributeSchemaRepository) FindByTenant(_ context.Context, tenantID string) (*groupdomain.TenantGroupAttributeSchema, error) {
	sharedmem.DefaultScope(&tenantID, kernel.DefaultTenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if schema := r.byTenant[tenantID]; schema != nil {
		return cloneGroupAttributeSchema(schema), nil
	}
	return nil, nil
}

func (r *TenantGroupAttributeSchemaRepository) Save(_ context.Context, schema *groupdomain.TenantGroupAttributeSchema) error {
	cloned := cloneGroupAttributeSchema(schema)
	sharedmem.DefaultScope(&cloned.TenantID, kernel.DefaultTenantID)
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.byTenant[cloned.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.byTenant[cloned.TenantID] = cloned
	return nil
}

func (r *TenantGroupAttributeSchemaRepository) Delete(_ context.Context, tenantID string) error {
	sharedmem.DefaultScope(&tenantID, kernel.DefaultTenantID)
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.byTenant, tenantID)
	return nil
}

// cloneGroupAttributeSchema は呼び出し側との aliasing を断つための深いコピー。
func cloneGroupAttributeSchema(s *groupdomain.TenantGroupAttributeSchema) *groupdomain.TenantGroupAttributeSchema {
	cloned := *s
	cloned.Attributes = slices.Clone(s.Attributes)
	return &cloned
}
