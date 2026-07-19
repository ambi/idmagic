package memory

import (
	"context"
	"slices"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedmem "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
)

// McpResourceServerRepository (ADR-055)

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
	return &cloned
}

func (r *McpResourceServerRepository) ListByTenant(_ context.Context, tenantID string) ([]*domain.McpResourceServer, error) {
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
	m := r.byID[sharedmem.TenantKey(tenantID, resourceServerID)]
	if m == nil {
		return nil, nil
	}
	return cloneMcpResourceServer(m), nil
}

func (r *McpResourceServerRepository) FindByResource(_ context.Context, tenantID, resource string) (*domain.McpResourceServer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	m := r.byRes[sharedmem.TenantKey(tenantID, resource)]
	if m == nil {
		return nil, nil
	}
	return cloneMcpResourceServer(m), nil
}

func (r *McpResourceServerRepository) Save(_ context.Context, m *domain.McpResourceServer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sharedmem.DefaultTenant(&m.TenantID)
	cloned := cloneMcpResourceServer(m)
	r.byID[sharedmem.TenantKey(m.TenantID, m.ResourceServerID)] = cloned
	r.byRes[sharedmem.TenantKey(m.TenantID, m.Resource)] = cloned
	return nil
}

func (r *McpResourceServerRepository) Delete(_ context.Context, tenantID, resourceServerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := sharedmem.TenantKey(tenantID, resourceServerID)
	m := r.byID[key]
	if m == nil {
		return nil
	}
	delete(r.byID, key)
	delete(r.byRes, sharedmem.TenantKey(tenantID, m.Resource))
	return nil
}
