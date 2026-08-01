package usecases

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// TenantAttributeSchemaRepo resolves a tenant's custom attribute schema. Satisfied
// structurally by Tenancy's ports.TenantUserAttributeSchemaRepository, without an
// import dependency on the Tenancy context.
type TenantAttributeSchemaRepo interface {
	FindByTenant(ctx context.Context, tenantID string) (*userdomain.TenantUserAttributeSchema, error)
}

// ResolveTenantAttributeDefs merges builtin attribute definitions with a tenant's
// custom schema (if any). The result is the attribute_defs input IssueClaimsWithFloor
// uses to enforce the visibility floor (ADR-151). A nil repo yields builtin defs only.
func ResolveTenantAttributeDefs(ctx context.Context, tenantID string, repo TenantAttributeSchemaRepo) ([]userdomain.UserAttributeDef, error) {
	defs := userdomain.BuiltinUserAttributeDefs()
	if repo == nil {
		return defs, nil
	}
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema != nil {
		defs = append(defs, schema.Attributes...)
	}
	return defs, nil
}
