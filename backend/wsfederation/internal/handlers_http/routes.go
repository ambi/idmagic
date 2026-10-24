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
	SecondFactors SecondFactorSteps

	WsFedRPRepo          wsfederationports.WsFedRelyingPartyRepository
	UserRepo             userports.UserRepository
	FederationSigner     samltoken.SignerProvider
	MessageIDReplayStore wsfederationports.MessageIDReplayStore
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
	// Negotiate は Authentication の任意の認証方式を試す。未構成なら nil である。
	Negotiate func(*echo.Context, string, bool) (bool, error)
}

// SecondFactorSteps は、Entra が MFA を求めた要求で、既存のセッションに第二要素の照合を求める操作である。
// 対話のログインを所有する Authentication が満たし、組み立て地点が結ぶ。
type SecondFactorSteps interface {
	SecondFactorMethods(c *echo.Context, sub string) []string
	// RequireSecondFactor はセッションを第二要素の照合の待ちにして Cookie を発行する。
	// セッションが失効していれば nil を返す。
	RequireSecondFactor(ctx context.Context, c *echo.Context, sessionID string) (*authdomain.AuthenticationContext, error)
	PendingPath(c *echo.Context, authn *authdomain.AuthenticationContext) string
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
