package ports

import (
	"context"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// TenantBrandingRepository は、Tenant の解決とは独立に更新する外装を保持する。
type TenantBrandingRepository interface {
	// FindByTenant は tenant の branding を返す。未設定なら nil, nil。
	FindByTenant(ctx context.Context, tenantID string) (*tenancydomain.TenantBranding, error)
	Save(ctx context.Context, branding *tenancydomain.TenantBranding) error
	// Update は最新の値の取得から change の適用と保存までを一つの排他区間で行う。
	// 未保存なら change に nil を渡す。change は外部への作用を行わない。
	Update(ctx context.Context, tenantID string, change BrandingChange) (*tenancydomain.TenantBranding, error)
}

type BrandingChange func(current *tenancydomain.TenantBranding) (*tenancydomain.TenantBranding, error)

// TenantBrandingAssetStore は branding ロゴ / favicon blob の保存境界。TenantBranding は
// object_key だけを持ち、binary 本体はこの store が所有する (と同型)。
type TenantBrandingAssetStore interface {
	Save(ctx context.Context, asset *tenancydomain.TenantBrandingAsset) error
	Find(ctx context.Context, tenantID string, kind tenancydomain.TenantBrandingAssetKind, objectKey string) (*tenancydomain.TenantBrandingAsset, error)
	DeleteByTenant(ctx context.Context, tenantID string, kind tenancydomain.TenantBrandingAssetKind) error
}
