// /.well-known/openid-configuration + /jwks
package handlers_http

import (
	"net/http"
	"sync"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/oauth2/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

var discoveryCache sync.Map // tenantID -> map[string]any

func (d Deps) handleDiscovery(c *echo.Context) error {
	if d.SCL == nil {
		return writeOAuthError(c, usecases.NewOAuthError("server_error", "SCL not loaded"))
	}
	tenantID := support.RequestTenantID(c)
	doc, err := d.SCL.BuildDiscoveryDocument(support.RequestIssuer(c, d.Issuer))
	if err != nil {
		return writeOAuthError(c, err)
	}
	// RFC 9396 — テナントで Enabled な authorization_details type を広告する (ADR-050)。
	if d.AuthzDetailTypeRepo != nil {
		types, err := d.AuthzDetailTypeRepo.ListByTenant(c.Request().Context(), tenantID)
		if err != nil {
			// DBエラー時はメモリキャッシュからフォールバック
			if cachedDoc, ok := discoveryCache.Load(tenantID); ok {
				if docMap, ok := cachedDoc.(map[string]any); ok {
					return c.JSON(http.StatusOK, docMap)
				}
			}
			return writeOAuthError(c, err)
		}
		supported := make([]string, 0, len(types))
		for _, t := range types {
			if t.State == oauthdomain.DetailTypeEnabled {
				supported = append(supported, t.Type)
			}
		}
		if len(supported) > 0 {
			doc["authorization_details_types_supported"] = supported
		}
	}

	// 成功したのでキャッシュを更新
	discoveryCache.Store(tenantID, doc)
	return c.JSON(http.StatusOK, doc)
}
