// Package http: tenancy コンテキストの HTTP アダプタ。
//
// テナント設定・ユーザ属性スキーマ・テナント CRUD (control-plane) のハンドラを所有し、
// 共有基盤 support.Deps を受け取って shared/handlers_http/server から登録される。
package handlers_http

import (
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

// Deps は tenancy HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	TenantRepo               tenantports.TenantRepository
	AttrSchemaRepo           tenantports.TenantUserAttributeSchemaRepository
	GroupAttrSchemaRepo      tenantports.TenantGroupAttributeSchemaRepository
	BrandingRepo             tenantports.TenantBrandingRepository
	BrandingAssetStore       tenantports.TenantBrandingAssetStore
	NotificationTemplateRepo tenantports.NotificationTemplateRepository
	// Notifier はテンプレート編集画面からのテスト送信に使う (wi-288)。
	Notifier  notificationports.Notifier
	UserRepo  userports.UserRepository
	GroupRepo groupports.GroupRepository
	QuotaRepo tenantports.QuotaRepository
	// FederationSigner resolves the request tenant's active public XML credential.
	FederationSigner samltoken.SignerProvider
}

// RegisterRoutes はテナント解決済みグループに、テナント単位の admin 設定・
// ユーザ属性スキーマ・branding のエンドポイントを登録する。branding の閲覧系
// (GetTenantBranding / GetTenantBrandingAsset) は未認証の login 画面等が読むため
// public とする (wi-89)。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/settings", d.handleGetAdminSettings)
	g.GET("/api/admin/v1/integration-endpoints", d.handleGetAdminIntegrationEndpoints)
	g.PATCH("/api/admin/v1/settings", d.handleUpdateAdminSettings)
	g.GET("/api/admin/v1/tenant/user_attribute_schema", d.handleGetUserAttributeSchema)
	g.PUT("/api/admin/v1/tenant/user_attribute_schema", d.handleUpdateUserAttributeSchema)
	g.GET("/api/admin/v1/tenant/group_attribute_schema", d.handleGetGroupAttributeSchema)
	g.PUT("/api/admin/v1/tenant/group_attribute_schema", d.handleUpdateGroupAttributeSchema)
	g.GET("/api/branding", d.handleGetBranding)
	g.PUT("/api/admin/v1/tenant/branding", d.handleUpdateBranding)
	g.POST("/api/admin/v1/tenant/branding/assets/:kind", d.handleUploadBrandingAsset)
	g.DELETE("/api/admin/v1/tenant/branding/assets/:kind", d.handleDeleteBrandingAsset)
	g.GET("/tenant-branding-assets/:kind/:id", d.handleGetBrandingAsset)
	g.GET("/api/admin/v1/tenant/notification_templates", d.handleListNotificationTemplates)
	g.GET("/api/admin/v1/tenant/notification_templates/:template_key/:locale", d.handleGetNotificationTemplate)
	g.PUT("/api/admin/v1/tenant/notification_templates/:template_key/:locale", d.handleUpdateNotificationTemplate)
	g.DELETE("/api/admin/v1/tenant/notification_templates/:template_key/:locale", d.handleResetNotificationTemplate)
	g.POST("/api/admin/v1/tenant/notification_templates/:template_key/:locale/preview", d.handlePreviewNotificationTemplate)
	g.POST("/api/admin/v1/tenant/notification_templates/:template_key/:locale/test", d.handleSendTestNotification)
}

// RegisterControlPlaneRoutes はテナント CRUD (system_admin 専用のテナント横断操作)
// を登録する。パスは他の admin API と揃えて `/api/admin/v1/tenants` とする (dev proxy /
// リバースプロキシは `/api` 配下を IdP へ転送する)。共有のテナント汎用グループ
// (/realms/:tenant_id) にそのまま登録し、default テナントへの限定は
// requireSystemAdmin (user.TenantID == DefaultTenantID) が担う。
// パス上の `:target_tenant_id` は CRUD 対象のテナント ID であり、グループ側の
// `:tenant_id` (リクエスト自身の realm) とは別物 — 同名にすると echo の
// Context.Param が外側の値を返してしまうため名前を分けている。
func RegisterControlPlaneRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/v1/tenants", d.handleListTenants)
	g.GET("/api/admin/v1/tenants/:target_tenant_id", d.handleGetTenant)
	g.POST("/api/admin/v1/tenants", d.handleCreateTenant)
	g.PATCH("/api/admin/v1/tenants/:target_tenant_id", d.handleUpdateTenant)
	g.PUT("/api/admin/v1/tenants/:target_tenant_id/endpoint_style", d.handleSetTenantEndpointStyle)
	g.POST("/api/admin/v1/tenants/:target_tenant_id/disable", d.handleDisableTenant)
	g.POST("/api/admin/v1/tenants/:target_tenant_id/enable", d.handleEnableTenant)
	g.PUT("/api/admin/v1/tenants/:target_tenant_id/quota", d.handleUpdateTenantQuota)
}
