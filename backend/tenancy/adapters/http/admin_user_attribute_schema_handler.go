package http

import (
	"errors"
	"net/http"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

type UserAttributeSchemaResponse struct {
	TenantID   string                        `json:"tenant_id"`
	Attributes []userdomain.UserAttributeDef `json:"attributes"`
	Builtin    []userdomain.UserAttributeDef `json:"builtin"`
	CreatedAt  time.Time                     `json:"created_at"`
	UpdatedAt  time.Time                     `json:"updated_at"`
}

type userAttributeSchemaUpdateRequest struct {
	Attributes []userdomain.UserAttributeDef `json:"attributes"`
}

func toUserAttributeSchemaResponse(schema *userdomain.TenantUserAttributeSchema) UserAttributeSchemaResponse {
	attributes := schema.Attributes
	if attributes == nil {
		attributes = []userdomain.UserAttributeDef{}
	}
	return UserAttributeSchemaResponse{
		TenantID:   schema.TenantID,
		Attributes: attributes,
		Builtin:    userdomain.BuiltinUserAttributeDefs(),
		CreatedAt:  schema.CreatedAt,
		UpdatedAt:  schema.UpdatedAt,
	}
}

func (d Deps) handleGetUserAttributeSchema(c *echo.Context) error {
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	schema, err := tenantusecases.GetUserAttributeSchema(c.Request().Context(), d.AttrSchemaRepo, actor.TenantID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toUserAttributeSchemaResponse(schema))
}

func (d Deps) handleUpdateUserAttributeSchema(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input userAttributeSchemaUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if d.GroupRepo != nil {
		existing, findErr := d.AttrSchemaRepo.FindByTenant(c.Request().Context(), actor.TenantID)
		if findErr != nil {
			return findErr
		}
		nextTypes := make(map[string]idmdomain.AttributeType, len(input.Attributes))
		for _, def := range input.Attributes {
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
		rules, listErr := d.GroupRepo.ListDynamicRules(c.Request().Context(), actor.TenantID)
		if listErr != nil {
			return listErr
		}
		for _, rule := range rules {
			for _, key := range rule.ReferencedAttributes {
				if changed[key] {
					return support.WriteBrowserError(c, http.StatusConflict, "attribute_referenced_by_dynamic_group", "動的グループのルールが参照している属性は削除または型変更できません")
				}
			}
		}
	}
	schema, err := tenantusecases.UpdateUserAttributeSchema(
		c.Request().Context(), d.AttrSchemaRepo, actor.TenantID, input.Attributes, time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, tenantusecases.ErrInvalidUserAttributeSchema) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_attribute_schema", "属性定義が不正です")
		}
		return err
	}
	if d.Emit != nil {
		keys := make([]string, len(schema.Attributes))
		for i, def := range schema.Attributes {
			keys[i] = def.Key
		}
		d.Emit(&tenancydomain.TenantUserAttributeSchemaUpdated{
			At: time.Now().UTC(), ActorUserID: actor.ID, TenantID: actor.TenantID, AttributeKeys: keys,
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, toUserAttributeSchemaResponse(schema))
}
