package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// TenantBrandingRepository はテナント単位の hosted UI ブランディング設定を保持する
// (wi-89, ADR-096)。Tenant aggregate には埋め込まず、TenantUserAttributeSchemaRepository
// と同じ理由で独立 entity として持つ。
type TenantBrandingRepository interface {
	// FindByTenant は tenant の branding を返す。未設定なら nil, nil。
	FindByTenant(ctx context.Context, tenantID string) (*domain.TenantBranding, error)
	Save(ctx context.Context, branding *domain.TenantBranding) error
}

// TenantBrandingAssetStore は branding ロゴ / favicon blob の保存境界。TenantBranding は
// object_key だけを持ち、binary 本体はこの store が所有する (ADR-096、ADR-073 と同型)。
type TenantBrandingAssetStore interface {
	Save(ctx context.Context, asset *domain.TenantBrandingAsset) error
	Find(ctx context.Context, tenantID string, kind domain.TenantBrandingAssetKind, objectKey string) (*domain.TenantBrandingAsset, error)
	DeleteByTenant(ctx context.Context, tenantID string, kind domain.TenantBrandingAssetKind) error
}
