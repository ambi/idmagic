package usecases

import (
	"context"
	"errors"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// ErrInvalidUserAttributeSchema は custom 属性定義が不正 (key 衝突 / 重複 / 形式違反)
// のときに返す。
var ErrInvalidUserAttributeSchema = errors.New("invalid attribute schema")

// GetUserAttributeSchema は tenant の custom 属性定義を返す。未定義のテナントには
// 空集合の schema を返し、呼び出し側が常に non-nil を扱えるようにする。
func GetUserAttributeSchema(
	ctx context.Context, repo tenantports.TenantUserAttributeSchemaRepository, tenantID string,
) (*tenancydomain.TenantUserAttributeSchema, error) {
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		now := time.Now().UTC()
		return &tenancydomain.TenantUserAttributeSchema{TenantID: tenantID, Attributes: []tenancydomain.UserAttributeDef{}, CreatedAt: now, UpdatedAt: now}, nil
	}
	return schema, nil
}

// UpdateUserAttributeSchema は tenant の custom 属性定義を全置換する。各定義を検証し、
// 組み込み属性との key 衝突・重複 key を拒否したうえで保存する。
func UpdateUserAttributeSchema(
	ctx context.Context, repo tenantports.TenantUserAttributeSchemaRepository,
	tenantID string, defs []tenancydomain.UserAttributeDef, now time.Time,
) (*tenancydomain.TenantUserAttributeSchema, error) {
	if defs == nil {
		defs = []tenancydomain.UserAttributeDef{}
	}
	schema := &tenancydomain.TenantUserAttributeSchema{
		TenantID:   tenantID,
		Attributes: defs,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	existing, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if existing != nil && !existing.CreatedAt.IsZero() {
		schema.CreatedAt = existing.CreatedAt
	}
	if err := repo.Validate(schema); err != nil {
		return nil, errors.Join(ErrInvalidUserAttributeSchema, err)
	}
	if err := repo.Save(ctx, schema); err != nil {
		return nil, err
	}
	return schema, nil
}
