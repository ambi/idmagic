package db_memory

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpToolCatalogRepository は登録ごとの最新のカタログをメモリに置く。
type McpToolCatalogRepository struct {
	mu       sync.RWMutex
	catalogs map[string]*domain.McpToolCatalog // key: catalogKey(tenant_id, resource_server_id)
}

func NewMcpToolCatalogRepository() *McpToolCatalogRepository {
	return &McpToolCatalogRepository{catalogs: map[string]*domain.McpToolCatalog{}}
}

func cloneMcpToolCatalog(catalog *domain.McpToolCatalog) *domain.McpToolCatalog {
	cloned := *catalog
	cloned.Tools = cloneCatalogTools(catalog.Tools)
	if catalog.Previous != nil {
		cloned.Previous = &domain.CatalogToolList{Tools: cloneCatalogTools(catalog.Previous.Tools), Complete: catalog.Previous.Complete}
	}
	return &cloned
}

func cloneCatalogTools(tools []domain.CatalogTool) []domain.CatalogTool {
	cloned := slices.Clone(tools)
	for index := range cloned {
		cloned[index].Arguments = slices.Clone(cloned[index].Arguments)
	}
	return cloned
}

func (r *McpToolCatalogRepository) Find(_ context.Context, tenantID, resourceServerID string) (*domain.McpToolCatalog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	catalog, ok := r.catalogs[catalogKey(tenantID, resourceServerID)]
	if !ok {
		return nil, nil
	}
	return cloneMcpToolCatalog(catalog), nil
}

// Save はカタログを置き換え、内容が保存済みのカタログと異なるときだけ保存済みの一覧を変更前として残す。
// 比較と置き換えを同じロックの中で行い、並行するカタログが変更前の一覧を失わないようにする。
func (r *McpToolCatalogRepository) Save(_ context.Context, catalog *domain.McpToolCatalog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := catalogKey(catalog.TenantID, catalog.ResourceServerID)
	saved := cloneMcpToolCatalog(catalog)
	saved.Previous, saved.ChangedAt = nil, time.Time{}
	if stored, ok := r.catalogs[key]; ok {
		if stored.Fingerprint() == catalog.Fingerprint() {
			saved.Previous, saved.ChangedAt = stored.Previous, stored.ChangedAt
		} else {
			saved.Previous = &domain.CatalogToolList{Tools: stored.Tools, Complete: stored.Complete}
			saved.ChangedAt = catalog.ReceivedAt
		}
	}
	r.catalogs[key] = cloneMcpToolCatalog(saved)
	return nil
}

func (r *McpToolCatalogRepository) Delete(_ context.Context, tenantID, resourceServerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.catalogs, catalogKey(tenantID, resourceServerID))
	return nil
}

func catalogKey(tenantID, resourceServerID string) string {
	return sharedmem.ScopedKey(tenantID, resourceServerID, kernel.DefaultTenantID)
}
