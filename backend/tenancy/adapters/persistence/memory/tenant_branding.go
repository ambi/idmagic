package memory

import (
	"context"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// =====================================================================
// TenantBrandingRepository (wi-89, ADR-096)
// =====================================================================

type TenantBrandingRepository struct {
	mu       sync.RWMutex
	byTenant map[string]*domain.TenantBranding
}

func NewTenantBrandingRepository() *TenantBrandingRepository {
	return &TenantBrandingRepository{byTenant: map[string]*domain.TenantBranding{}}
}

func (r *TenantBrandingRepository) FindByTenant(_ context.Context, tenantID string) (*domain.TenantBranding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if branding := r.byTenant[tenantID]; branding != nil {
		cloned := *branding
		return &cloned, nil
	}
	return nil, nil
}

func (r *TenantBrandingRepository) Save(_ context.Context, branding *domain.TenantBranding) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := *branding
	if existing := r.byTenant[branding.TenantID]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	r.byTenant[branding.TenantID] = &cloned
	return nil
}

// =====================================================================
// TenantBrandingAssetStore (wi-89, ADR-096)
// =====================================================================

type TenantBrandingAssetStore struct {
	mu     sync.RWMutex
	assets map[string]*domain.TenantBrandingAsset // key: tenant_id + kind + object_key
}

func NewTenantBrandingAssetStore() *TenantBrandingAssetStore {
	return &TenantBrandingAssetStore{assets: map[string]*domain.TenantBrandingAsset{}}
}

func brandingAssetKey(tenantID string, kind domain.TenantBrandingAssetKind, objectKey string) string {
	return strings.Join([]string{tenantID, string(kind), objectKey}, "\x00")
}

func cloneBrandingAsset(asset *domain.TenantBrandingAsset) *domain.TenantBrandingAsset {
	cloned := *asset
	cloned.Data = append([]byte(nil), asset.Data...)
	return &cloned
}

func (s *TenantBrandingAssetStore) Save(_ context.Context, asset *domain.TenantBrandingAsset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := brandingAssetKey(asset.TenantID, asset.Kind, asset.ObjectKey)
	cloned := cloneBrandingAsset(asset)
	if existing := s.assets[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	s.assets[key] = cloned
	return nil
}

func (s *TenantBrandingAssetStore) Find(_ context.Context, tenantID string, kind domain.TenantBrandingAssetKind, objectKey string) (*domain.TenantBrandingAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asset := s.assets[brandingAssetKey(tenantID, kind, objectKey)]
	if asset == nil {
		return nil, nil
	}
	return cloneBrandingAsset(asset), nil
}

func (s *TenantBrandingAssetStore) DeleteByTenant(_ context.Context, tenantID string, kind domain.TenantBrandingAssetKind) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	prefix := tenantID + "\x00" + string(kind) + "\x00"
	for key := range s.assets {
		if strings.HasPrefix(key, prefix) {
			delete(s.assets, key)
		}
	}
	return nil
}
