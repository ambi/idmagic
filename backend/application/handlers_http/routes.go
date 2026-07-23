// Package http は Application bounded context の HTTP アダプタ (wi-69)。
//
// 運用者向け Application カタログ (CRUD・protocol binding・割当) と、利用者ポータル向けの
// 割当済みアプリ一覧を所有する。共有基盤 support.Deps を受け取り、shared/handlers_http/server から
// tenant 解決済みグループに登録される。
package handlers_http

import (
	appports "github.com/ambi/idmagic/backend/application/ports"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"

	"github.com/labstack/echo/v5"
)

// Deps は Application HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator

	ApplicationRepo             appports.ApplicationRepository
	ApplicationIconStore        appports.ApplicationIconStore
	ApplicationAssignmentRepo   appports.AssignmentRepository
	ApplicationOrderingRepo     appports.ApplicationOrderingRepository
	ApplicationCategoryRepo     appports.ApplicationCategoryRepository
	ApplicationSignInPolicyRepo appports.SignInPolicyRepository
	DefaultSignInPolicyRepo     appports.DefaultSignInPolicyRepository
	GroupRepo                   groupports.GroupRepository
	UserRepo                    userports.UserRepository
	ClientRepo                  oauthports.OAuth2ClientRepository
	WsFedRPRepo                 wsfederationports.WsFedRelyingPartyRepository
	SamlSPRepo                  samlports.SamlServiceProviderRepository
	// ProvisioningNotifier is the outbound Provisioning boundary port (wi-45,
	// ADR-128). nil means outbound provisioning is not wired.
	ProvisioningNotifier appports.ProvisioningNotifier
	// QuotaRepo enforces the tenant's Hard Quota on applications (wi-160,
	// ADR-134). nil skips enforcement.
	QuotaRepo tenantports.QuotaRepository
}

// RegisterRoutes は Application カタログの admin / account エンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/api/admin/applications", d.handleListApplications)
	g.POST("/api/admin/applications", d.handleCreateApplication)
	g.GET("/api/admin/applications/:application_id", d.handleGetApplication)
	g.PATCH("/api/admin/applications/:application_id", d.handleUpdateApplication)
	g.DELETE("/api/admin/applications/:application_id", d.handleDeleteApplication)
	g.POST("/api/admin/applications/:application_id/icon", d.handleUploadApplicationIcon)
	g.DELETE("/api/admin/applications/:application_id/icon", d.handleDeleteApplicationIcon)
	g.POST("/api/admin/applications/:application_id/bindings", d.handleAttachBinding)
	g.DELETE("/api/admin/applications/:application_id/bindings/:binding_type", d.handleDetachBinding)
	g.PATCH("/api/admin/applications/:application_id/oidc", d.handleUpdateOIDCConfig)
	g.POST("/api/admin/applications/:application_id/oidc/rotate-secret", d.handleRotateOIDCClientSecret)
	g.PATCH("/api/admin/applications/:application_id/wsfed", d.handleUpdateWsFedConfig)
	g.PATCH("/api/admin/applications/:application_id/saml", d.handleUpdateSamlConfig)
	g.GET("/api/admin/applications/:application_id/assignments", d.handleListAssignments)
	g.POST("/api/admin/applications/:application_id/assignments", d.handleAssignApplication)
	g.DELETE("/api/admin/applications/:application_id/assignments/:subject_type/:subject_id", d.handleUnassignApplication)
	g.GET("/api/admin/applications/:application_id/sign-in-policy", d.handleGetSignInPolicy)
	g.PUT("/api/admin/applications/:application_id/sign-in-policy", d.handleUpdateSignInPolicy)
	g.GET("/api/admin/default-sign-in-policy", d.handleGetDefaultSignInPolicy)
	g.PUT("/api/admin/default-sign-in-policy", d.handleUpdateDefaultSignInPolicy)
	g.PUT("/api/admin/applications/:application_id/categories", d.handleSetApplicationCategories)
	g.GET("/api/admin/application-categories", d.handleListCategories)
	g.POST("/api/admin/application-categories", d.handleCreateCategory)
	g.PATCH("/api/admin/application-categories/:category_id", d.handleUpdateCategory)
	g.DELETE("/api/admin/application-categories/:category_id", d.handleDeleteCategory)
	g.GET("/api/account/applications", d.handleListMyApplications)
	g.GET("/api/account/applications/order", d.handleGetMyApplicationOrder)
	g.PUT("/api/account/applications/order", d.handleReorderMyApplications)
	g.GET("/application-icons/:application_id/:object_key", d.handleGetApplicationIcon)
}
