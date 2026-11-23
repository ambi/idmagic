package db_memory

import (
	"context"
	"strings"
	"sync"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// =====================================================================
// TenantBrandingRepository (wi-89)
// =====================================================================

type TenantBrandingRepository struct {
	mu       sync.RWMutex
	byTenant map[string]*tenancydomain.TenantBranding
}

func NewTenantBrandingRepository() *TenantBrandingRepository {
	return &TenantBrandingRepository{byTenant: map[string]*tenancydomain.TenantBranding{}}
}

func (r *TenantBrandingRepository) FindByTenant(_ context.Context, tenantID string) (*tenancydomain.TenantBranding, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if branding := r.byTenant[tenantID]; branding != nil {
		cloned := *branding
		return &cloned, nil
	}
	return nil, nil
}

func (r *TenantBrandingRepository) Save(ctx context.Context, branding *tenancydomain.TenantBranding) error {
	_, err := r.Update(ctx, branding.TenantID, func(_ *tenancydomain.TenantBranding) (*tenancydomain.TenantBranding, error) {
		return branding, nil
	})
	return err
}

func (r *TenantBrandingRepository) Update(_ context.Context, tenantID string, change tenantports.BrandingChange) (*tenancydomain.TenantBranding, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var current *tenancydomain.TenantBranding
	if existing := r.byTenant[tenantID]; existing != nil {
		cloned := *existing
		current = &cloned
	}
	updated, err := change(current)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, nil
	}
	cloned := *updated
	if current != nil && !current.CreatedAt.IsZero() {
		cloned.CreatedAt = current.CreatedAt
	}
	r.byTenant[tenantID] = &cloned
	result := cloned
	return &result, nil
}

// =====================================================================
// TenantBrandingAssetStore (wi-89)
// =====================================================================

type TenantBrandingAssetStore struct {
	mu     sync.RWMutex
	assets map[string]*tenancydomain.TenantBrandingAsset // key: tenant_id + kind + object_key
}

func NewTenantBrandingAssetStore() *TenantBrandingAssetStore {
	return &TenantBrandingAssetStore{assets: map[string]*tenancydomain.TenantBrandingAsset{}}
}

func brandingAssetKey(tenantID string, kind tenancydomain.TenantBrandingAssetKind, objectKey string) string {
	return strings.Join([]string{tenantID, string(kind), objectKey}, "\x00")
}

func cloneBrandingAsset(asset *tenancydomain.TenantBrandingAsset) *tenancydomain.TenantBrandingAsset {
	cloned := *asset
	cloned.Data = append([]byte(nil), asset.Data...)
	return &cloned
}

func (s *TenantBrandingAssetStore) Save(_ context.Context, asset *tenancydomain.TenantBrandingAsset) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := brandingAssetKey(asset.TenantID, asset.Kind, asset.ID)
	cloned := cloneBrandingAsset(asset)
	if existing := s.assets[key]; existing != nil && !existing.CreatedAt.IsZero() {
		cloned.CreatedAt = existing.CreatedAt
	}
	s.assets[key] = cloned
	return nil
}

func (s *TenantBrandingAssetStore) Find(_ context.Context, tenantID string, kind tenancydomain.TenantBrandingAssetKind, objectKey string) (*tenancydomain.TenantBrandingAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	asset := s.assets[brandingAssetKey(tenantID, kind, objectKey)]
	if asset == nil {
		return nil, nil
	}
	return cloneBrandingAsset(asset), nil
}

func (s *TenantBrandingAssetStore) DeleteByTenant(_ context.Context, tenantID string, kind tenancydomain.TenantBrandingAssetKind) error {
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
