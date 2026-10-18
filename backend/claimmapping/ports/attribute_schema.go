// Package ports は ClaimMapping がほかのモジュールに求める契約を置く。
package ports

import (
	"context"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// TenantAttributeSchemaRepo はテナントのカスタム属性定義を読む。
// IdManagement の保存ポートが構造的に満たし、Tenancy の管理用の表現には依存しない。
type TenantAttributeSchemaRepo interface {
	FindByTenant(ctx context.Context, tenantID string) (*userdomain.TenantUserAttributeSchema, error)
}

// ResolveTenantAttributeDefs reads the tenant's custom schema through repo and merges it
// with the builtin definitions. A nil repo yields builtin defs only.
func ResolveTenantAttributeDefs(ctx context.Context, tenantID string, repo TenantAttributeSchemaRepo) ([]userdomain.UserAttributeDef, error) {
	if repo == nil {
		return claimdomain.MergeTenantAttributeDefs(nil), nil
	}
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return claimdomain.MergeTenantAttributeDefs(schema), nil
}
