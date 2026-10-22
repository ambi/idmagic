package ports

import (
	"context"
	"errors"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// ErrAttributeValuesNotUnique は、unique に変えようとした属性の値を、テナントの削除されていない
// 二人以上の User が持つので、Save がスキーマを変えなかったことを表す。
var ErrAttributeValuesNotUnique = errors.New("attribute values are not unique")

// TenantUserAttributeSchemaRepository は tenant ごとの custom attribute 定義集合
// を保持する。tenant aggregate には埋め込まず独立 aggregate として
// 持ち、tenant 削除時に Delete で cascade する。
type TenantUserAttributeSchemaRepository interface {
	// FindByTenant は tenant の schema を返す。未定義なら nil, nil。
	FindByTenant(ctx context.Context, tenantID string) (*domain.TenantUserAttributeSchema, error)
	Save(ctx context.Context, schema *domain.TenantUserAttributeSchema) error
	Delete(ctx context.Context, tenantID string) error
	Validate(schema *domain.TenantUserAttributeSchema) error
	Builtin() []domain.UserAttributeDef
}
