package db_memory

import (
	"context"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// =====================================================================
// TenantUserAttributeSchemaRepository (wi-19)
// =====================================================================

type TenantUserAttributeSchemaRepository struct {
	mu       sync.RWMutex
	byTenant map[string]*userdomain.TenantUserAttributeSchema
	// users は、unique にする属性の既存の値を確かめる User の保存先。nil なら確かめない。
	users *UserRepository
}

func NewTenantUserAttributeSchemaRepository() *TenantUserAttributeSchemaRepository {
	return &TenantUserAttributeSchemaRepository{byTenant: map[string]*userdomain.TenantUserAttributeSchema{}}
}

func (r *TenantUserAttributeSchemaRepository) FindByTenant(_ context.Context, tenantID string) (*userdomain.TenantUserAttributeSchema, error) {
	sharedmem.DefaultScope(&tenantID, kernel.DefaultTenantID)
	r.mu.RLock()
	defer r.mu.RUnlock()
	if schema := r.byTenant[tenantID]; schema != nil {
		return cloneUserAttributeSchema(schema), nil
	}
	return nil, nil
}

func (r *TenantUserAttributeSchemaRepository) Save(_ context.Context, schema *userdomain.TenantUserAttributeSchema) error {
	cloned := cloneUserAttributeSchema(schema)
	sharedmem.DefaultScope(&cloned.TenantID, kernel.DefaultTenantID)
	// User の保存と同じく users.mu、r.mu の順に取り、確かめてから保存するまでの間の保存を止める。
	if r.users != nil {
		r.users.mu.Lock()
		defer r.users.mu.Unlock()
		if err := r.users.firstSharedLocked(cloned.TenantID, cloned.UniqueKeys(), nil, func(key string) error {
			return &userports.AttributeValuesNotUniqueError{Key: key}
		}); err != nil {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.byTenant[cloned.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.byTenant[cloned.TenantID] = cloned
	return nil
}

func (r *TenantUserAttributeSchemaRepository) Delete(_ context.Context, tenantID string) error {
	sharedmem.DefaultScope(&tenantID, kernel.DefaultTenantID)
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
