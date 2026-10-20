// Package http は WsFederation モジュールの HTTP アダプタ (wi-61)。
//
// WS-Federation passive requestor profile のブラウザエンドポイントを所有する。
// 共有基盤 support.Deps を受け取り、shared/handlers_http/server から tenant 解決済みグループに登録される。
package handlers_http

import (
	"context"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	claimports "github.com/ambi/idmagic/backend/claimmapping/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"

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
	// Passwords は UsernameToken を、対話のログインと同じ試行の制限と監査で認証する。
	// Authentication が所有し、組み立て地点が結ぶ。
	Passwords      signinports.PasswordAuthenticator
	AttrSchemaRepo claimports.TenantAttributeSchemaRepo
}

// Sessions は、パッシブサインインが読み、サインアウトが失効させるログインセッションである。
// Authentication が所有し、組み立て地点が結ぶ。
type Sessions struct {
	AuthnResolver  authdomain.AuthenticationContextResolver
	SessionManager SessionRevoker
}

// SessionRevoker はログアウトで、cookie が指すログインセッションを失効させる。Authentication の
// SessionManager が満たし、組み立て地点が渡す。
type SessionRevoker interface {
	Revoke(ctx context.Context, cookieHeader string) error
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
