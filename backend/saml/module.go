// Package saml は SAML bounded context の DI 組立を所有する (ADR-091)。
package saml

import (
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	samlhttp "github.com/ambi/idmagic/backend/saml/adapters/http"
	"github.com/ambi/idmagic/backend/saml/ports"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/wsfederation/adapters/samltoken"

	"github.com/labstack/echo/v5"
)

type Module struct {
	SPRepo ports.SamlServiceProviderRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	applicationGate *support.ApplicationGate, userRepo idmports.UserRepository, federationSigner *samltoken.Signer,
) {
	samlhttp.RegisterRoutes(g, samlhttp.Deps{
		Deps: deps, Authenticator: authenticator, ApplicationGate: applicationGate,
		SamlSPRepo: m.SPRepo, FederationSigner: federationSigner, UserRepo: userRepo,
	})
}
