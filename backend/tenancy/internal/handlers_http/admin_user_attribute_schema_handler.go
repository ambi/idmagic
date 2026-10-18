package handlers_http

import (
	"errors"
	"net/http"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/internal/usecases"

	"github.com/labstack/echo/v5"
)

type UserAttributeSchemaResponse struct {
	TenantID   string                           `json:"tenant_id"`
	Attributes []tenancydomain.UserAttributeDef `json:"attributes"`
	Builtin    []tenancydomain.UserAttributeDef `json:"builtin"`
	CreatedAt  time.Time                        `json:"created_at"`
	UpdatedAt  time.Time                        `json:"updated_at"`
}

type userAttributeSchemaUpdateRequest struct {
	Attributes []tenancydomain.UserAttributeDef `json:"attributes"`
}

func (d Deps) toUserAttributeSchemaResponse(schema *tenancydomain.TenantUserAttributeSchema) UserAttributeSchemaResponse {
	attributes := schema.Attributes
	if attributes == nil {
		attributes = []tenancydomain.UserAttributeDef{}
	}
	return UserAttributeSchemaResponse{
		TenantID:   schema.TenantID,
		Attributes: attributes,
		Builtin:    d.AttrSchemaRepo.Builtin(),
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
	return support.NoStoreJSON(c, http.StatusOK, d.toUserAttributeSchemaResponse(schema))
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if d.GroupAttributeReferences != nil {
		existing, findErr := d.AttrSchemaRepo.FindByTenant(c.Request().Context(), actor.TenantID)
		if findErr != nil {
			return findErr
		}
		nextTypes := make(map[string]string, len(input.Attributes))
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
		references, listErr := d.GroupAttributeReferences(c.Request().Context(), actor.TenantID)
		if listErr != nil {
			return listErr
		}
		for _, attributes := range references {
			for _, key := range attributes {
				if changed[key] {
					return support.WriteProblem(c, http.StatusConflict, "attribute_referenced_by_dynamic_group", "An attribute referenced by a dynamic group rule cannot be deleted or have its type changed.")
				}
			}
		}
	}
	schema, err := tenantusecases.UpdateUserAttributeSchema(
		c.Request().Context(), d.AttrSchemaRepo, actor.TenantID, input.Attributes, time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, tenantusecases.ErrInvalidUserAttributeSchema) {
			return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_attribute_schema", "The attribute definition is invalid.")
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
	return support.NoStoreJSON(c, http.StatusOK, d.toUserAttributeSchemaResponse(schema))
}
