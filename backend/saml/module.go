// Package saml は SAML bounded context の DI 組立を所有する (ADR-091)。
package saml

import (
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	samlhttp "github.com/ambi/idmagic/backend/saml/handlers_http"
	"github.com/ambi/idmagic/backend/saml/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

type Module struct {
	SPRepo      ports.SamlServiceProviderRepository
	ReplayStore ports.AuthnRequestReplayStore
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	applicationGate *support.ApplicationGate, userRepo userports.UserRepository, federationSigner *samltoken.Signer,
) {
	samlhttp.RegisterRoutes(g, samlhttp.Deps{
		Deps: deps, Authenticator: authenticator, ApplicationGate: applicationGate,
		SamlSPRepo: m.SPRepo, ReplayStore: m.ReplayStore, FederationSigner: federationSigner, UserRepo: userRepo,
	})
}
