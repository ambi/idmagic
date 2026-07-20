// Package http: tenancy コンテキストの HTTP アダプタ。
//
// テナント設定・ユーザ属性スキーマ・テナント CRUD (control-plane) のハンドラを所有し、
// 共有基盤 support.Deps を受け取って shared/handlers_http/server から登録される。
package handlers_http

import (
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// Deps は tenancy HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	TenantRepo         tenantports.TenantRepository
	AttrSchemaRepo     tenantports.TenantUserAttributeSchemaRepository
	BrandingRepo       tenantports.TenantBrandingRepository
	BrandingAssetStore tenantports.TenantBrandingAssetStore
	UserRepo           userports.UserRepository
	GroupRepo          groupports.GroupRepository
}

// RegisterRoutes はテナント解決済みグループに、テナント単位の admin 設定・
// ユーザ属性スキーマ・branding のエンドポイントを登録する。branding の閲覧系
// (GetTenantBranding / GetTenantBrandingAsset) は未認証の login 画面等が読むため
// public とする (wi-89, ADR-096)。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/settings", d.handleGetAdminSettings)
	g.PATCH("/api/admin/settings", d.handleUpdateAdminSettings)
	g.GET("/api/admin/tenant/user_attribute_schema", d.handleGetUserAttributeSchema)
	g.PUT("/api/admin/tenant/user_attribute_schema", d.handleUpdateUserAttributeSchema)
	g.GET("/api/branding", d.handleGetBranding)
	g.PUT("/api/admin/tenant/branding", d.handleUpdateBranding)
	g.POST("/api/admin/tenant/branding/assets/:kind", d.handleUploadBrandingAsset)
	g.DELETE("/api/admin/tenant/branding/assets/:kind", d.handleDeleteBrandingAsset)
	g.GET("/tenant-branding-assets/:kind/:object_key", d.handleGetBrandingAsset)
}

// RegisterControlPlaneRoutes はテナント CRUD (system_admin 専用 of テナント横断操作)
// を登録する。パスは他の admin API と揃えて `/api/admin/tenants` とする (dev proxy /
// リバースプロキシは `/api` 配下を IdP へ転送する)。control-plane グループ
// (/realms/default 配下、ADR-032) と bare グループの両方から呼ばれる。
func RegisterControlPlaneRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/tenants", d.handleListTenants)
	g.GET("/api/admin/tenants/:tenant_id", d.handleGetTenant)
	g.POST("/api/admin/tenants", d.handleCreateTenant)
	g.PATCH("/api/admin/tenants/:tenant_id", d.handleUpdateTenant)
	g.POST("/api/admin/tenants/:tenant_id/disable", d.handleDisableTenant)
	g.POST("/api/admin/tenants/:tenant_id/enable", d.handleEnableTenant)
}
