package usecases

import (
	"context"
	"errors"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
)

// ErrInvalidGroupAttributeSchema は Group custom 属性定義が不正 (型不正、キー重複) の
// ときに返す。
var ErrInvalidGroupAttributeSchema = errors.New("invalid attribute schema")

// GetGroupAttributeSchema は tenant の Group custom 属性定義を返す。未定義のテナントには
// 空集合の schema を返し、呼び出し側が常に non-nil を扱えるようにする。
func GetGroupAttributeSchema(
	ctx context.Context, repo groupports.TenantGroupAttributeSchemaRepository, tenantID string,
) (*groupdomain.TenantGroupAttributeSchema, error) {
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return &groupdomain.TenantGroupAttributeSchema{TenantID: tenantID, Attributes: []groupdomain.GroupAttributeDef{}}, nil
	}
	return schema, nil
}

// UpdateGroupAttributeSchema は tenant の Group custom 属性定義を全置換する。各定義を
// 検証し、重複 key を拒否したうえで保存する。
func UpdateGroupAttributeSchema(
	ctx context.Context, repo groupports.TenantGroupAttributeSchemaRepository,
	tenantID string, defs []groupdomain.GroupAttributeDef, now time.Time, checks ...func(*groupdomain.TenantGroupAttributeSchema) error,
) (*groupdomain.TenantGroupAttributeSchema, error) {
	if defs == nil {
		defs = []groupdomain.GroupAttributeDef{}
	}
	schema := &groupdomain.TenantGroupAttributeSchema{
		TenantID:   tenantID,
		Attributes: defs,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := schema.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidGroupAttributeSchema, err)
	}
	return repo.Update(ctx, tenantID, func(existing *groupdomain.TenantGroupAttributeSchema) (*groupdomain.TenantGroupAttributeSchema, error) {
		current := existing
		if current == nil {
			current = &groupdomain.TenantGroupAttributeSchema{TenantID: tenantID, Attributes: []groupdomain.GroupAttributeDef{}}
		}
		for _, check := range checks {
			if err := check(current); err != nil {
				return nil, err
			}
		}
		if existing != nil && !existing.CreatedAt.IsZero() {
			schema.CreatedAt = existing.CreatedAt
		}
		return schema, nil
	})
}
