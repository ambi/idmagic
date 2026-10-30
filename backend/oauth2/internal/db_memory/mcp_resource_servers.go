package db_memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpResourceServerRepository

type McpResourceServerRepository struct {
	mu    sync.RWMutex
	byID  map[string]*domain.McpResourceServer // key: sharedmem.TenantKey(tenant_id, resource_server_id)
	byRes map[string]*domain.McpResourceServer // key: sharedmem.TenantKey(tenant_id, resource)
}

func NewMcpResourceServerRepository() *McpResourceServerRepository {
	return &McpResourceServerRepository{
		byID:  map[string]*domain.McpResourceServer{},
		byRes: map[string]*domain.McpResourceServer{},
	}
}

// Seed は起動時のサンプル登録投入に使う (テスト・デモ用)。
func (r *McpResourceServerRepository) Seed(m *domain.McpResourceServer) {
	_ = r.Save(context.Background(), m)
}

func cloneMcpResourceServer(m *domain.McpResourceServer) *domain.McpResourceServer {
	cloned := *m
	cloned.Scopes = slices.Clone(m.Scopes)
	cloned.AccessRules = slices.Clone(m.AccessRules)
	return &cloned
}

func (r *McpResourceServerRepository) ListAll(_ context.Context, tenantID string) ([]*domain.McpResourceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.McpResourceServer, 0)
	for _, m := range r.byID {
		if m.TenantID == tenantID {
			out = append(out, cloneMcpResourceServer(m))
		}
	}
	slices.SortFunc(out, func(a, b *domain.McpResourceServer) int { return strings.Compare(a.Resource, b.Resource) })
	return out, nil
}

func (r *McpResourceServerRepository) FindByID(_ context.Context, tenantID, resourceServerID string) (*domain.McpResourceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.byID[sharedmem.ScopedKey(tenantID, resourceServerID, kernel.DefaultTenantID)]
	if m == nil {
		return nil, nil
	}
	return cloneMcpResourceServer(m), nil
}

func (r *McpResourceServerRepository) FindByResource(_ context.Context, tenantID, resource string) (*domain.McpResourceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.byRes[sharedmem.ScopedKey(tenantID, resource, kernel.DefaultTenantID)]
	if m == nil {
		return nil, nil
	}
	return cloneMcpResourceServer(m), nil
}

func (r *McpResourceServerRepository) FindByClientID(_ context.Context, tenantID, clientID string) (*domain.McpResourceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.byID {
		if m.TenantID == tenantID && clientID != "" && m.ClientID == clientID {
			return cloneMcpResourceServer(m), nil
		}
	}
	return nil, nil
}

func (r *McpResourceServerRepository) Save(_ context.Context, m *domain.McpResourceServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sharedmem.DefaultScope(&m.TenantID, kernel.DefaultTenantID)
	cloned := cloneMcpResourceServer(m)
	r.byID[sharedmem.ScopedKey(m.TenantID, m.ID, kernel.DefaultTenantID)] = cloned
	r.byRes[sharedmem.ScopedKey(m.TenantID, m.Resource, kernel.DefaultTenantID)] = cloned
	return nil
}

func (r *McpResourceServerRepository) Delete(_ context.Context, tenantID, resourceServerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.ScopedKey(tenantID, resourceServerID, kernel.DefaultTenantID)
	m := r.byID[key]
	if m == nil {
		return nil
	}
	delete(r.byID, key)
	delete(r.byRes, sharedmem.ScopedKey(tenantID, m.Resource, kernel.DefaultTenantID))
	return nil
}
