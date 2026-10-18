package usecases

import (
	"context"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
)

// TenantGroupCSVSchemaReader は IdManagement の属性定義の保存先を Group CSV の照会ポートへ接続する。
// Group に組み込みの属性はなく、未定義のスキーマは追加の列を返さない。
type TenantGroupCSVSchemaReader struct {
	Repository groupports.TenantGroupAttributeSchemaRepository
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
