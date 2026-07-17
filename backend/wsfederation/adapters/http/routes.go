// Package http は WsFederation bounded context の HTTP アダプタ (wi-61)。
//
// WS-Federation passive requestor profile のブラウザエンドポイントを所有する。
// 共有基盤 support.Deps を受け取り、shared/adapters/http/server から tenant 解決済みグループに登録される。
package http

import (
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"
	wsfederationports "github.com/ambi/idmagic/backend/wsfederation/ports"

	"github.com/labstack/echo/v5"
)

// Deps は WS-Federation HTTP ハンドラが必要とする依存。
type Deps struct {
	support.Deps
	*support.Authenticator
	*support.ApplicationGate

	WsFedRPRepo                wsfederationports.WsFedRelyingPartyRepository
	UserRepo                   idmports.UserRepository
	FederationSigner           *samltoken.Signer
	ClientAssertionReplayStore oauthports.ClientAssertionReplayStore
	LoginAttemptThrottle       authnports.LoginAttemptThrottle
	PasswordHasher             authnports.PasswordHasher
	SentinelPasswordHash       string
}

// RegisterRoutes は WS-Federation passive のエンドポイントを登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.GET("/wsfed", d.handleWsFed)
	g.GET("/federationmetadata/2007-06/federationmetadata.xml", d.handleFederationMetadata)
	g.GET("/trust/mex", d.handleTrustMEX)
	g.POST("/trust/usernamemixed", d.handleWsTrustUsernameMixed)
	g.GET("/api/admin/wsfed/relying-parties", d.handleListRelyingParties)
	g.POST("/api/admin/wsfed/relying-parties", d.handleUpsertRelyingParty)
	g.DELETE("/api/admin/wsfed/relying-parties", d.handleDeleteRelyingParty)
	g.POST("/api/admin/wsfed/entra-federation", d.handleConfigureEntraFederation)
}
