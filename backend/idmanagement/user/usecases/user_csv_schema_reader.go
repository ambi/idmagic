package usecases

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// TenantUserCSVSchemaReader adapts the Tenancy-owned schema repository to the
// narrow inward-facing User CSV port.
type TenantUserCSVSchemaReader struct {
	Repository tenantports.TenantUserAttributeSchemaRepository
}

func (r TenantUserCSVSchemaReader) EffectiveUserAttributeDefs(ctx context.Context, tenantID string) ([]userdomain.UserAttributeDef, error) {
	if r.Repository == nil {
		return nil, nil
	}
	schema, err := r.Repository.FindByTenant(ctx, tenantID)
	if err != nil || schema == nil {
		return nil, err
	}
	return append([]userdomain.UserAttributeDef(nil), schema.Attributes...), nil
}
