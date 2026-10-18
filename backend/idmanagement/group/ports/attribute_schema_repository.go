package ports

import (
	"context"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

// TenantGroupAttributeSchemaRepository は IdManagement が所有するカスタム属性定義の保存先である。
type TenantGroupAttributeSchemaRepository interface {
	FindByTenant(ctx context.Context, tenantID string) (*groupdomain.TenantGroupAttributeSchema, error)
	Save(ctx context.Context, schema *groupdomain.TenantGroupAttributeSchema) error
	Delete(ctx context.Context, tenantID string) error
}
