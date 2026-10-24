package handlers_http

import (
	"errors"
	"net/http"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

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

func handleGetGroupAttributeSchema(d Deps, c *echo.Context) error {
	actor, err := requireAttributeSchemaAdmin(d, c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	schema, err := groupusecases.GetGroupAttributeSchema(c.Request().Context(), d.GroupAttrSchemaRepo, actor.TenantID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toGroupAttributeSchemaResponse(schema))
}

func handleUpdateGroupAttributeSchema(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := requireAttributeSchemaAdmin(d, c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input groupAttributeSchemaUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	schema, err := groupusecases.UpdateGroupAttributeSchema(
		c.Request().Context(), d.GroupAttrSchemaRepo, actor.TenantID, input.Attributes, time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, groupusecases.ErrInvalidGroupAttributeSchema) {
			return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_attribute_schema", "The attribute definition is invalid.")
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
