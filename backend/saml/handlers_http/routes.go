// Package http は Saml bounded context の HTTP アダプタ (wi-29)。
//
// SAML 2.0 Web Browser SSO Profile のブラウザエンドポイント (metadata / SSO / SLO) と、
// service provider 管理 API を所有する。共有基盤 support.Deps を受け取り、shared/handlers_http/server から
// tenant 解決済みグループに登録される。
package handlers_http

import (
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

// Deps は SAML HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	*support.ApplicationGate

	SamlSPRepo       samlports.SamlServiceProviderRepository
	IDPProfileRepo   samlports.SamlIdentityProviderProfileRepository
	ReplayStore      samlports.AuthnRequestReplayStore
	FederationSigner samltoken.SignerProvider
	UserRepo         userports.UserRepository
	AttrSchemaRepo   claimusecases.TenantAttributeSchemaRepo
}

// RegisterRoutes は SAML 2.0 IdP のエンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/saml/metadata", d.handleSamlMetadata)
	g.GET("/saml/signing-certificate.pem", d.handleSamlSigningCertificate)
	g.GET("/saml/sso", d.handleSamlSSORedirect)
	g.POST("/saml/sso", d.handleSamlSSOPost)
	g.GET("/saml/slo", d.handleSamlSLO)
	g.POST("/saml/slo", d.handleSamlSLO)
	g.GET("/saml/idp/:profile_id/metadata", d.handleSamlMetadata)
	g.GET("/saml/idp/:profile_id/signing-certificate.pem", d.handleSamlSigningCertificate)
	g.GET("/saml/idp/:profile_id/sso", d.handleSamlSSORedirect)
	g.POST("/saml/idp/:profile_id/sso", d.handleSamlSSOPost)
	g.GET("/saml/idp/:profile_id/slo", d.handleSamlSLO)
	g.POST("/saml/idp/:profile_id/slo", d.handleSamlSLO)
	g.GET("/api/admin/saml/service-providers", d.handleListServiceProviders)
	g.POST("/api/admin/saml/service-providers", d.handleUpsertServiceProvider)
	g.DELETE("/api/admin/saml/service-providers", d.handleDeleteServiceProvider)
	g.GET("/api/admin/saml/idp-profiles", d.handleListIDPProfiles)
	g.POST("/api/admin/saml/idp-profiles", d.handleCreateIDPProfile)
	g.PUT("/api/admin/saml/idp-profiles/:profile_id", d.handleUpdateIDPProfile)
	g.DELETE("/api/admin/saml/idp-profiles/:profile_id", d.handleDeleteIDPProfile)
}
