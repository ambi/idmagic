// /.well-known/openid-configuration + /jwks
package http

import (
	"net/http"
	"sync"

	"github.com/ambi/idmagic/internal/oauth2/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/spec"

	"github.com/labstack/echo/v5"
)

var (
	discoveryCache sync.Map // tenantID -> map[string]any
	jwksCache      sync.Map // tenantID -> []map[string]any
)

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
			if t.State == spec.DetailTypeEnabled {
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

func (d Deps) handleJWKS(c *echo.Context) error {
	tenantID := support.RequestTenantID(c)
	keys, err := d.KeyStore.GetAllKeys(c.Request().Context())
	if err != nil {
		// DBエラー時はメモリキャッシュからフォールバック
		if cachedKeys, ok := jwksCache.Load(tenantID); ok {
			if keysList, ok := cachedKeys.([]map[string]any); ok {
				return c.JSON(http.StatusOK, map[string]any{"keys": keysList})
			}
		}
		return writeOAuthError(c, err)
	}
	jwks := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		jwks = append(jwks, k.PublicJWK)
	}

	// 成功したのでキャッシュを更新
	jwksCache.Store(tenantID, jwks)
	return c.JSON(http.StatusOK, map[string]any{"keys": jwks})
}
