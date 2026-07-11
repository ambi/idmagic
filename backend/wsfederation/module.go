// Package wsfederation は WS-Federation bounded context の DI 組立を所有する (ADR-091)。
package wsfederation

import (
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	wsfedhttp "github.com/ambi/idmagic/backend/wsfederation/adapters/http"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"

	"github.com/labstack/echo/v5"
)

type Module struct {
	RPRepo wsfedports.WsFedRelyingPartyRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	applicationGate *support.ApplicationGate, userRepo idmports.UserRepository, federationSigner *samltoken.Signer,
	clientAssertionReplayStore oauthports.ClientAssertionReplayStore, loginAttemptThrottle authnports.LoginAttemptThrottle,
	passwordHasher authnports.PasswordHasher, sentinelPasswordHash string,
) {
	wsfedhttp.RegisterRoutes(g, wsfedhttp.Deps{
		Deps: deps, Authenticator: authenticator, ApplicationGate: applicationGate, WsFedRPRepo: m.RPRepo,
		UserRepo: userRepo, FederationSigner: federationSigner, ClientAssertionReplayStore: clientAssertionReplayStore,
		LoginAttemptThrottle: loginAttemptThrottle, PasswordHasher: passwordHasher, SentinelPasswordHash: sentinelPasswordHash,
	})
}
