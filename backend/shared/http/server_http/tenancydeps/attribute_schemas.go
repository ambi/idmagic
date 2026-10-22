// Package tenancydeps は Tenancy の要求する値と IdManagement の公開操作を組み立てる。
package tenancydeps

import (
	"context"
	"errors"
	"fmt"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// UserAttributeSchemas は保存と管理の表現を分け、定義の意味を IdManagement の検証に委ねる。
func UserAttributeSchemas(repo userports.TenantUserAttributeSchemaRepository) tenantports.TenantUserAttributeSchemaRepository {
	if repo == nil {
		return nil
	}
	return userAttributeSchemas{repo: repo}
}

type userAttributeSchemas struct {
	repo userports.TenantUserAttributeSchemaRepository
}

func (r userAttributeSchemas) FindByTenant(ctx context.Context, tenantID string) (*tenantdomain.TenantUserAttributeSchema, error) {
	schema, err := r.repo.FindByTenant(ctx, tenantID)
	if err != nil || schema == nil {
		return nil, err
	}
	return &tenantdomain.TenantUserAttributeSchema{
		TenantID: schema.TenantID, Attributes: userDefinitions(schema.Attributes),
		CreatedAt: schema.CreatedAt, UpdatedAt: schema.UpdatedAt,
	}, nil
}

func (r userAttributeSchemas) Save(ctx context.Context, schema *tenantdomain.TenantUserAttributeSchema) error {
	err := r.repo.Save(ctx, identityUserSchema(schema))
	if notUnique, ok := errors.AsType[*userports.AttributeValuesNotUniqueError](err); ok {
		return fmt.Errorf("%w: %s", tenantports.ErrAttributeValuesNotUnique, notUnique.Key)
	}
	return err
}

func (r userAttributeSchemas) Delete(ctx context.Context, tenantID string) error {
	return r.repo.Delete(ctx, tenantID)
}

func (r userAttributeSchemas) Validate(schema *tenantdomain.TenantUserAttributeSchema) error {
	return identityUserSchema(schema).Validate()
}

func (r userAttributeSchemas) Builtin() []tenantdomain.UserAttributeDef {
	return userDefinitions(userdomain.BuiltinUserAttributeDefs())
}

func identityUserSchema(schema *tenantdomain.TenantUserAttributeSchema) *userdomain.TenantUserAttributeSchema {
	var definitions []userdomain.UserAttributeDef
	if schema.Attributes != nil {
		definitions = make([]userdomain.UserAttributeDef, len(schema.Attributes))
		for i, def := range schema.Attributes {
			definitions[i] = userdomain.UserAttributeDef{
				Key: def.Key, Label: def.Label, Type: idmdomain.AttributeType(def.Type),
				MultiValued: def.MultiValued, Required: def.Required, EditableByUser: def.EditableByUser,
				Unique:    def.Unique,
				ClaimName: cloneString(def.ClaimName), OIDCScope: cloneString(def.OIDCScope),
				Visibility: idmdomain.AttrVisibility(def.Visibility), PII: def.PII,
			}
		}
	}
	return &userdomain.TenantUserAttributeSchema{
		TenantID: schema.TenantID, Attributes: definitions,
		CreatedAt: schema.CreatedAt, UpdatedAt: schema.UpdatedAt,
	}
}

func userDefinitions(definitions []userdomain.UserAttributeDef) []tenantdomain.UserAttributeDef {
	if definitions == nil {
		return nil
	}
	converted := make([]tenantdomain.UserAttributeDef, len(definitions))
	for i, def := range definitions {
		converted[i] = tenantdomain.UserAttributeDef{
			Key: def.Key, Label: def.Label, Type: string(def.Type),
			MultiValued: def.MultiValued, Required: def.Required, EditableByUser: def.EditableByUser,
			Unique:    def.Unique,
			ClaimName: cloneString(def.ClaimName), OIDCScope: cloneString(def.OIDCScope),
			Visibility: string(def.Visibility), PII: def.PII,
		}
	}
	return converted
}

// GroupAttributeSchemas は Group の保存型を管理用の値へ変換する。
func GroupAttributeSchemas(repo groupports.TenantGroupAttributeSchemaRepository) tenantports.TenantGroupAttributeSchemaRepository {
	if repo == nil {
		return nil
	}
	return groupAttributeSchemas{repo: repo}
}

type groupAttributeSchemas struct {
	repo groupports.TenantGroupAttributeSchemaRepository
}

func (r groupAttributeSchemas) FindByTenant(ctx context.Context, tenantID string) (*tenantdomain.TenantGroupAttributeSchema, error) {
	schema, err := r.repo.FindByTenant(ctx, tenantID)
	if err != nil || schema == nil {
		return nil, err
	}
	var definitions []tenantdomain.GroupAttributeDef
	if schema.Attributes != nil {
		definitions = make([]tenantdomain.GroupAttributeDef, len(schema.Attributes))
		for i, def := range schema.Attributes {
			definitions[i] = tenantdomain.GroupAttributeDef{
				Key: def.Key, Label: def.Label, Type: string(def.Type), MultiValued: def.MultiValued, Required: def.Required,
			}
		}
	}
	return &tenantdomain.TenantGroupAttributeSchema{
		TenantID: schema.TenantID, Attributes: definitions, CreatedAt: schema.CreatedAt, UpdatedAt: schema.UpdatedAt,
	}, nil
}

func (r groupAttributeSchemas) Save(ctx context.Context, schema *tenantdomain.TenantGroupAttributeSchema) error {
	return r.repo.Save(ctx, identityGroupSchema(schema))
}

func (r groupAttributeSchemas) Delete(ctx context.Context, tenantID string) error {
	return r.repo.Delete(ctx, tenantID)
}

func (r groupAttributeSchemas) Validate(schema *tenantdomain.TenantGroupAttributeSchema) error {
	return identityGroupSchema(schema).Validate()
}

func identityGroupSchema(schema *tenantdomain.TenantGroupAttributeSchema) *groupdomain.TenantGroupAttributeSchema {
	var definitions []groupdomain.GroupAttributeDef
	if schema.Attributes != nil {
		definitions = make([]groupdomain.GroupAttributeDef, len(schema.Attributes))
		for i, def := range schema.Attributes {
			definitions[i] = groupdomain.GroupAttributeDef{
				Key: def.Key, Label: def.Label, Type: idmdomain.AttributeType(def.Type), MultiValued: def.MultiValued, Required: def.Required,
			}
		}
	}
	return &groupdomain.TenantGroupAttributeSchema{
		TenantID: schema.TenantID, Attributes: definitions, CreatedAt: schema.CreatedAt, UpdatedAt: schema.UpdatedAt,
	}
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	return new(*value)
}
