package ports

import (
	"context"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// TenantUserAttributeSchemaRepository は IdManagement が所有するカスタム属性定義の保存先である。
type TenantUserAttributeSchemaRepository interface {
	FindByTenant(ctx context.Context, tenantID string) (*userdomain.TenantUserAttributeSchema, error)
	Save(ctx context.Context, schema *userdomain.TenantUserAttributeSchema) error
	Delete(ctx context.Context, tenantID string) error
}
