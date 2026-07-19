// /.well-known/oauth-protected-resource (RFC 9728 Protected Resource Metadata, ADR-055)
package http

import (
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleProtectedResourceMetadata(c *echo.Context) error {
	resource := c.QueryParam("resource")
	if resource == "" {
		return writeOAuthError(c, usecases.NewOAuthError("invalid_request", "resource パラメータが必要です"))
	}
	tenantID := support.RequestTenantID(c)
	meta, err := usecases.BuildProtectedResourceMetadata(
		c.Request().Context(), d.McpResourceServerRepo, tenantID, resource, support.RequestIssuer(c, d.Issuer),
	)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if d.Emit != nil {
		d.Emit(&domain.ProtectedResourceMetadataServed{At: time.Now().UTC(), TenantID: tenantID, Resource: resource})
	}
	return c.JSON(http.StatusOK, meta)
}
