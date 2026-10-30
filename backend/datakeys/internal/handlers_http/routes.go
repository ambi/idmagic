// Package handlers_http owns the DataKeys HTTP bindings: the read-only health
// surface and the emergency rotate and disable actions. All of them are
// control-plane only; destroy has no HTTP binding.
package handlers_http

import (
	"github.com/ambi/idmagic/backend/datakeys/internal/ports"
	"github.com/ambi/idmagic/backend/datakeys/internal/usecases"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
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
	Cache      ports.CacheInvalidator
	Migrators  *usecases.MigratorRegistry
	Jobs       jobsports.Enqueuer
}

// パス上の `:target_tenant_id` は操作の対象のテナントであり、グループ側の `:tenant_id`
// （リクエスト自身の realm）とは別物である。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/data-keys/health", d.handleListTenantDataKeyHealth)
	g.POST("/api/admin/v1/tenants/:target_tenant_id/data-key/rotate", d.handleRotateTenantDataKey)
	g.POST("/api/admin/v1/tenants/:target_tenant_id/data-key/versions/:version/disable", d.handleDisableTenantDataKey)
}
