package usecases

import (
	"context"
	"errors"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
)

// ErrInvalidUserAttributeSchema は custom 属性定義が不正 (key 衝突 / 重複 / 形式違反)
// のときに返す。
var ErrInvalidUserAttributeSchema = errors.New("invalid attribute schema")

// ErrAttributeReferencedByDynamicGroup は、参照中の属性の削除と型変更の拒否を表す。
var ErrAttributeReferencedByDynamicGroup = errors.New("attribute referenced by dynamic group")

// GetUserAttributeSchema は tenant の custom 属性定義を返す。未定義のテナントには
// 空集合の schema を返し、呼び出し側が常に non-nil を扱えるようにする。
func GetUserAttributeSchema(
	ctx context.Context, repo userports.TenantUserAttributeSchemaRepository, tenantID string,
) (*userdomain.TenantUserAttributeSchema, error) {
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		now := time.Now().UTC()
		return &userdomain.TenantUserAttributeSchema{TenantID: tenantID, Attributes: []userdomain.UserAttributeDef{}, CreatedAt: now, UpdatedAt: now}, nil
	}
	return schema, nil
}

// UpdateUserAttributeSchema は tenant の custom 属性定義を全置換する。各定義を検証し、
// 組み込み属性との key 衝突・重複 key を拒否したうえで保存する。
func UpdateUserAttributeSchema(
	ctx context.Context, repo userports.TenantUserAttributeSchemaRepository, groups groupports.GroupRepository,
	tenantID string, defs []userdomain.UserAttributeDef, now time.Time,
) (*userdomain.TenantUserAttributeSchema, error) {
	if groups != nil {
		if err := protectReferencedUserAttributes(ctx, repo, groups, tenantID, defs); err != nil {
			return nil, err
		}
	}
	if defs == nil {
		defs = []userdomain.UserAttributeDef{}
	}
	schema := &userdomain.TenantUserAttributeSchema{
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
	if err := schema.Validate(); err != nil {
		return nil, errors.Join(ErrInvalidUserAttributeSchema, err)
	}
	if err := repo.Save(ctx, schema); err != nil {
		return nil, err
	}
	return schema, nil
}

func protectReferencedUserAttributes(
	ctx context.Context, repo userports.TenantUserAttributeSchemaRepository, groups groupports.GroupRepository,
	tenantID string, defs []userdomain.UserAttributeDef,
) error {
	existing, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	nextTypes := make(map[string]idmdomain.AttributeType, len(defs))
	for _, def := range defs {
		nextTypes[def.Key] = def.Type
	}
	changed := map[string]bool{}
	if existing != nil {
		for _, def := range existing.Attributes {
			if next, ok := nextTypes[def.Key]; !ok || next != def.Type {
				changed[def.Key] = true
			}
		}
	}
	rules, err := groups.ListDynamicRules(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, rule := range rules {
		for _, key := range rule.ReferencedAttributes {
			if changed[key] {
				return ErrAttributeReferencedByDynamicGroup
			}
		}
	}
	return nil
}
