package ports

import (
	"context"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

// TenantGroupAttributeSchemaRepository は tenant ごとの Group custom attribute 定義集合
// を保持する。tenant aggregate には埋め込まず独立 aggregate として持ち、tenant 削除時に
// Delete で cascade する。TenantUserAttributeSchemaRepository と同じ配置方針 (schema 管理は
// Tenancy コンテキストが所有する)。
type TenantGroupAttributeSchemaRepository interface {
	// FindByTenant は tenant の schema を返す。未定義なら nil, nil。
	FindByTenant(ctx context.Context, tenantID string) (*groupdomain.TenantGroupAttributeSchema, error)
	Save(ctx context.Context, schema *groupdomain.TenantGroupAttributeSchema) error
	Delete(ctx context.Context, tenantID string) error
}
