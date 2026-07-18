// Package http は Saml bounded context の HTTP アダプタ (wi-29)。
//
// SAML 2.0 Web Browser SSO Profile のブラウザエンドポイント (metadata / SSO / SLO) と、
// service provider 管理 API を所有する。共有基盤 support.Deps を受け取り、shared/adapters/http/server から
// tenant 解決済みグループに登録される。
package http

import (
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	samlports "github.com/ambi/idmagic/backend/saml/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"

	"github.com/labstack/echo/v5"
)

// Deps は SAML HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	*support.ApplicationGate

	SamlSPRepo       samlports.SamlServiceProviderRepository
	ReplayStore      samlports.AuthnRequestReplayStore
	FederationSigner *samltoken.Signer
	UserRepo         idmports.UserRepository
}

// RegisterRoutes は SAML 2.0 IdP のエンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/saml/metadata", d.handleSamlMetadata)
	g.GET("/saml/sso", d.handleSamlSSORedirect)
	g.POST("/saml/sso", d.handleSamlSSOPost)
	g.GET("/saml/slo", d.handleSamlSLO)
	g.POST("/saml/slo", d.handleSamlSLO)
	g.GET("/api/admin/saml/service-providers", d.handleListServiceProviders)
	g.POST("/api/admin/saml/service-providers", d.handleUpsertServiceProvider)
	g.DELETE("/api/admin/saml/service-providers", d.handleDeleteServiceProvider)
}
