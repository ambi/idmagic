// Package http は WsFederation モジュールの HTTP アダプタ (wi-61)。
//
// WS-Federation passive requestor profile のブラウザエンドポイントを所有する。
// 共有基盤 support.Deps を受け取り、shared/handlers_http/server から tenant 解決済みグループに登録される。
package handlers_http

import (
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

// Deps は WS-Federation HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	ApplicationGate
	Sessions

	WsFedRPRepo                wsfederationports.WsFedRelyingPartyRepository
	UserRepo                   userports.UserRepository
	FederationSigner           samltoken.SignerProvider
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	LoginAttemptThrottle       sessionports.LoginAttemptThrottle
	PasswordHasher             passwordports.PasswordHasher
	SentinelPasswordHash       string
	AttrSchemaRepo             claimusecases.TenantAttributeSchemaRepo
}

// Sessions は、パッシブサインインが読み、サインアウトが失効させるログインセッションである。
// Authentication が所有し、組み立て地点が結ぶ。
type Sessions struct {
	AuthnResolver  authdomain.AuthenticationContextResolver
	SessionManager *sessionusecases.SessionManager
}

// RegisterRoutes は WS-Federation passive のエンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/wsfed", d.handleWsFed)
	g.GET("/federationmetadata/2007-06/federationmetadata.xml", d.handleFederationMetadata)
	g.GET("/trust/mex", d.handleTrustMEX)
	g.POST("/trust/usernamemixed", d.handleWsTrustUsernameMixed)
	g.GET("/api/admin/v1/wsfed/relying-parties", d.handleListRelyingParties)
	g.POST("/api/admin/v1/wsfed/relying-parties", d.handleUpsertRelyingParty)
	g.DELETE("/api/admin/v1/wsfed/relying-parties", d.handleDeleteRelyingParty)
	g.POST("/api/admin/v1/wsfed/entra-federation", d.handleConfigureEntraFederation)
}
