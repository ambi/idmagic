package usecases

import (
	"context"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// TenantGroupCSVSchemaReader adapts the Tenancy-owned schema repository to the
// narrow inward-facing Group CSV port. Group has no builtin catalogue to union,
// so an undefined tenant schema resolves to no custom columns at all rather
// than to a default set.
type TenantGroupCSVSchemaReader struct {
	Repository tenantports.TenantGroupAttributeSchemaRepository
}

func (r TenantGroupCSVSchemaReader) EffectiveGroupAttributeDefs(ctx context.Context, tenantID string) ([]groupdomain.GroupAttributeDef, error) {
	if r.Repository == nil {
		return nil, nil
	}
	schema, err := r.Repository.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, nil
	}
	return schema.EffectiveDefs(), nil
}

var _ groupports.EffectiveGroupAttributeSchemaReader = TenantGroupCSVSchemaReader{}
