package handlers_http

import (
	"errors"
	"net/http"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

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

func handleGetUserAttributeSchema(d Deps, c *echo.Context) error {
	actor, err := requireAttributeSchemaAdmin(d, c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	schema, err := userusecases.GetUserAttributeSchema(c.Request().Context(), d.AttrSchemaRepo, actor.TenantID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toUserAttributeSchemaResponse(schema))
}

func handleUpdateUserAttributeSchema(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := requireAttributeSchemaAdmin(d, c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input userAttributeSchemaUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	schema, err := userusecases.UpdateUserAttributeSchema(
		c.Request().Context(), d.AttrSchemaRepo, d.GroupRepo, actor.TenantID, input.Attributes, time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, userusecases.ErrAttributeReferencedByDynamicGroup) {
			return support.WriteProblem(c, http.StatusConflict, "attribute_referenced_by_dynamic_group", "An attribute referenced by a dynamic group rule cannot be deleted or have its type or write authority changed.")
		}
		if errors.Is(err, userusecases.ErrInvalidUserAttributeSchema) {
			return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_attribute_schema", "The attribute definition is invalid.")
		}
		if _, ok := errors.AsType[*userports.AttributeValuesNotUniqueError](err); ok {
			return support.WriteProblem(c, http.StatusConflict, "attribute_values_not_unique", "Users share a value of an attribute that would become unique.")
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
