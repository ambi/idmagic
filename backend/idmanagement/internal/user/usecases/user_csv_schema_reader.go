package usecases

import (
	"context"

	userschemaports "github.com/ambi/idmagic/backend/idmanagement/user/ports"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// TenantUserCSVSchemaReader は IdManagement の属性定義の保存先を User CSV の照会ポートへ接続する。
type TenantUserCSVSchemaReader struct {
	Repository userschemaports.TenantUserAttributeSchemaRepository
}

func (r TenantUserCSVSchemaReader) EffectiveUserAttributeDefs(ctx context.Context, tenantID string) ([]userdomain.UserAttributeDef, error) {
	if r.Repository == nil {
		return nil, nil
	}
	schema, err := r.Repository.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return userdomain.BuiltinUserAttributeDefs(), nil
	}
	return schema.EffectiveDefs(), nil
}
