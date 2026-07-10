package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

// =====================================================================
// AuthorizationDetailTypeRepository (RFC 9396 / ADR-050)
// =====================================================================

type AuthorizationDetailTypeRepository struct {
	mu    sync.RWMutex
	types map[string]*oauthdomain.AuthorizationDetailType // key: TenantKey(tenant_id, type)
}

func NewAuthorizationDetailTypeRepository() *AuthorizationDetailTypeRepository {
	return &AuthorizationDetailTypeRepository{types: map[string]*oauthdomain.AuthorizationDetailType{}}
}

// Seed は起動時のサンプル type 投入に使う (テスト・デモ用)。
func (r *AuthorizationDetailTypeRepository) Seed(t *oauthdomain.AuthorizationDetailType) {
	_ = r.Save(context.Background(), t)
}

func cloneDetailType(t *oauthdomain.AuthorizationDetailType) *oauthdomain.AuthorizationDetailType {
	cloned := *t
	cloned.Schema.Rules = slices.Clone(t.Schema.Rules)
	for i := range cloned.Schema.Rules {
		cloned.Schema.Rules[i].Allowed = slices.Clone(t.Schema.Rules[i].Allowed)
	}
	return &cloned
}

func (r *AuthorizationDetailTypeRepository) ListByTenant(_ context.Context, tenantID string) ([]*oauthdomain.AuthorizationDetailType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*oauthdomain.AuthorizationDetailType, 0)
	for _, t := range r.types {
		if t.TenantID == tenantID {
			out = append(out, cloneDetailType(t))
		}
	}
	slices.SortFunc(out, func(a, b *oauthdomain.AuthorizationDetailType) int { return strings.Compare(a.Type, b.Type) })
	return out, nil
}

func (r *AuthorizationDetailTypeRepository) FindByType(_ context.Context, tenantID, detailType string) (*oauthdomain.AuthorizationDetailType, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t := r.types[TenantKey(tenantID, detailType)]
	if t == nil {
		return nil, nil
	}
	return cloneDetailType(t), nil
}

func (r *AuthorizationDetailTypeRepository) Save(_ context.Context, t *oauthdomain.AuthorizationDetailType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	defaultTenant(&t.TenantID)
	r.types[TenantKey(t.TenantID, t.Type)] = cloneDetailType(t)
	return nil
}

func (r *AuthorizationDetailTypeRepository) Delete(_ context.Context, tenantID, detailType string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.types, TenantKey(tenantID, detailType))
	return nil
}
