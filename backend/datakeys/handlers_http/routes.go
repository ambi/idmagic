// Package handlers_http owns the DataKeys HTTP bindings (wi-97 T007): a
// read-only, system_admin-gated health surface, matching the minimal admin
// footprint the work item's Scope calls for (no rotate/disable/destroy
// action endpoints).
package handlers_http

import (
	"github.com/ambi/idmagic/backend/datakeys/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/envelope_crypto"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

type Deps struct {
	support.Deps
	*support.Authenticator
	Repository ports.DataKeyRepository
	Crypto     envelope_crypto.EnvelopeCrypto
	TenantRepo tenantports.TenantRepository
}

func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/data-keys/health", d.handleListTenantDataKeyHealth)
}
