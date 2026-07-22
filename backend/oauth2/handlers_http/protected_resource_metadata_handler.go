// /.well-known/oauth-protected-resource (RFC 9728 Protected Resource Metadata, ADR-055)
package handlers_http

import (
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleProtectedResourceMetadata(c *echo.Context) error {
	resource := c.QueryParam("resource")
	tenantID := support.RequestTenantID(c)
	meta, err := tokenusecases.BuildProtectedResourceMetadata(
		c.Request().Context(), d.McpResourceServerRepo, tenantID, resource, support.RequestIssuer(c, d.Issuer),
	)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if d.Emit != nil {
		eventResource := resource
		if eventResource == "" {
			eventResource = meta.Resource
		}
		d.Emit(&domain.ProtectedResourceMetadataServed{At: time.Now().UTC(), TenantID: tenantID, Resource: eventResource})
	}
	return c.JSON(http.StatusOK, meta)
}
