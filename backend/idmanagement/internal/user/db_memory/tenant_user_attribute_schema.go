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

func (r *TenantUserAttributeSchemaRepository) Save(ctx context.Context, schema *userdomain.TenantUserAttributeSchema) error {
	_, err := r.Update(ctx, schema.TenantID, func(_ *userdomain.TenantUserAttributeSchema) (*userdomain.TenantUserAttributeSchema, error) {
		return schema, nil
	})
	return err
}

func (r *TenantUserAttributeSchemaRepository) Update(_ context.Context, tenantID string, change func(*userdomain.TenantUserAttributeSchema) (*userdomain.TenantUserAttributeSchema, error)) (*userdomain.TenantUserAttributeSchema, error) {
	sharedmem.DefaultScope(&tenantID, kernel.DefaultTenantID)
	// User の保存と同じく users.mu、r.mu の順に取り、確かめてから保存するまでの間の保存を止める。
	if r.users != nil {
		r.users.mu.Lock()
		defer r.users.mu.Unlock()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var current *userdomain.TenantUserAttributeSchema
	if stored := r.byTenant[tenantID]; stored != nil {
		current = cloneUserAttributeSchema(stored)
	}
	updated, err := change(current)
	if err != nil {
		return nil, err
	}
	cloned := cloneUserAttributeSchema(updated)
	cloned.TenantID = tenantID
	if r.users != nil {
		if err := r.users.firstSharedLocked(tenantID, cloned.UniqueKeys(), nil, func(key string) error { return &userports.AttributeValuesNotUniqueError{Key: key} }); err != nil {
			return nil, err
		}
	}
	if existing := r.byTenant[tenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.byTenant[tenantID] = cloned
	return cloneUserAttributeSchema(cloned), nil
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
