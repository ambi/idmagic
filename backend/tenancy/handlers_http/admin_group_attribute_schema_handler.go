package handlers_http

import (
	"errors"
	"net/http"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

type GroupAttributeSchemaResponse struct {
	TenantID   string                          `json:"tenant_id"`
	Attributes []groupdomain.GroupAttributeDef `json:"attributes"`
	CreatedAt  time.Time                       `json:"created_at"`
	UpdatedAt  time.Time                       `json:"updated_at"`
}

type groupAttributeSchemaUpdateRequest struct {
	Attributes []groupdomain.GroupAttributeDef `json:"attributes"`
}

func toGroupAttributeSchemaResponse(schema *groupdomain.TenantGroupAttributeSchema) GroupAttributeSchemaResponse {
	attributes := schema.Attributes
	if attributes == nil {
		attributes = []groupdomain.GroupAttributeDef{}
	}
	return GroupAttributeSchemaResponse{
		TenantID:   schema.TenantID,
		Attributes: attributes,
		CreatedAt:  schema.CreatedAt,
		UpdatedAt:  schema.UpdatedAt,
	}
}

func (d Deps) handleGetGroupAttributeSchema(c *echo.Context) error {
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	schema, err := tenantusecases.GetGroupAttributeSchema(c.Request().Context(), d.GroupAttrSchemaRepo, actor.TenantID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toGroupAttributeSchemaResponse(schema))
}

func (d Deps) handleUpdateGroupAttributeSchema(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.requireTenantAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input groupAttributeSchemaUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	schema, err := tenantusecases.UpdateGroupAttributeSchema(
		c.Request().Context(), d.GroupAttrSchemaRepo, actor.TenantID, input.Attributes, time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, tenantusecases.ErrInvalidGroupAttributeSchema) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_attribute_schema", "The attribute definition is invalid.")
		}
		return err
	}
	if d.Emit != nil {
		keys := make([]string, len(schema.Attributes))
		for i, def := range schema.Attributes {
			keys[i] = def.Key
		}
		d.Emit(&tenancydomain.TenantGroupAttributeSchemaUpdated{
			At: time.Now().UTC(), ActorUserID: actor.ID, TenantID: actor.TenantID, AttributeKeys: keys,
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, toGroupAttributeSchemaResponse(schema))
}
