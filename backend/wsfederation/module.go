// Package wsfederation は WS-Federation bounded context の DI 組立を所有する (ADR-091)。
package wsfederation

import (
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	sessionports "github.com/ambi/idmagic/backend/authentication/session/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	wsfedhttp "github.com/ambi/idmagic/backend/wsfederation/handlers_http"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

type Module struct {
	RPRepo wsfedports.WsFedRelyingPartyRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	applicationGate *support.ApplicationGate, userRepo userports.UserRepository, federationSigner *samltoken.Signer,
	clientAssertionReplayStore oauthports.ClientAssertionReplayStore, loginAttemptThrottle sessionports.LoginAttemptThrottle,
	passwordHasher passwordports.PasswordHasher, sentinelPasswordHash string,
) {
	wsfedhttp.RegisterRoutes(g, wsfedhttp.Deps{
		Deps: deps, Authenticator: authenticator, ApplicationGate: applicationGate, WsFedRPRepo: m.RPRepo,
		UserRepo: userRepo, FederationSigner: federationSigner, ClientAssertionReplayStore: clientAssertionReplayStore,
		LoginAttemptThrottle: loginAttemptThrottle, PasswordHasher: passwordHasher, SentinelPasswordHash: sentinelPasswordHash,
	})
}
