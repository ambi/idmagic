package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// TenantBrandingRepository は、Tenant の解決とは独立に更新する外装を保持する。
type TenantBrandingRepository interface {
	// FindByTenant は tenant の branding を返す。未設定なら nil, nil。
	FindByTenant(ctx context.Context, tenantID string) (*domain.TenantBranding, error)
	Save(ctx context.Context, branding *domain.TenantBranding) error
}

// TenantBrandingAssetStore は branding ロゴ / favicon blob の保存境界。TenantBranding は
// object_key だけを持ち、binary 本体はこの store が所有する (と同型)。
type TenantBrandingAssetStore interface {
	Save(ctx context.Context, asset *domain.TenantBrandingAsset) error
	Find(ctx context.Context, tenantID string, kind domain.TenantBrandingAssetKind, objectKey string) (*domain.TenantBrandingAsset, error)
	DeleteByTenant(ctx context.Context, tenantID string, kind domain.TenantBrandingAssetKind) error
}
