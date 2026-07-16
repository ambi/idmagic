// Package http owns SigningKeys HTTP bindings while preserving existing paths.
package http

import (
	"net/http"
	"sync"

	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/signingkeys/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	KeyStore   ports.KeyStore
	TenantRepo tenantports.TenantRepository
}

var jwksCache sync.Map

func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/jwks", d.handleJWKS)
	g.GET("/api/admin/keys", d.handleListAdminKeys)
	g.GET("/api/admin/keys/health", d.handleListTenantKeyHealth)
	g.GET("/api/admin/keys/:kid", d.handleGetAdminKey)
	g.POST("/api/admin/keys/rotate", d.handleRotateTenantKey)
	g.POST("/api/admin/keys/:kid/disable", d.handleDisableTenantKey)
}

func (d Deps) handleJWKS(c *echo.Context) error {
	tenantID := support.RequestTenantID(c)
	keys, err := d.KeyStore.GetAllKeys(c.Request().Context())
	if err != nil {
		if cachedKeys, ok := jwksCache.Load(tenantID); ok {
			if keysList, ok := cachedKeys.([]map[string]any); ok {
				return c.JSON(http.StatusOK, map[string]any{"keys": keysList})
			}
		}
		return err
	}
	jwks := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		jwks = append(jwks, key.PublicJWK)
	}
	jwksCache.Store(tenantID, jwks)
	return c.JSON(http.StatusOK, map[string]any{"keys": jwks})
}
