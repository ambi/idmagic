// Package ports は ClaimMapping がほかのモジュールに求める契約を置く。
package ports

import (
	"context"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// TenantAttributeSchemaRepo resolves a tenant's custom attribute schema. Satisfied
// structurally by Tenancy's ports.TenantUserAttributeSchemaRepository, without an
// import dependency on the Tenancy context.
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
