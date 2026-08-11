// Package saml は SAML bounded context の DI 組立を所有する。
package saml

import (
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	samlhttp "github.com/ambi/idmagic/backend/saml/handlers_http"
	"github.com/ambi/idmagic/backend/saml/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/labstack/echo/v5"
)

type Module struct {
	SPRepo      ports.SamlServiceProviderRepository
	ProfileRepo ports.SamlIdentityProviderProfileRepository
	ReplayStore ports.AuthnRequestReplayStore
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	applicationGate *support.ApplicationGate, userRepo userports.UserRepository, federationSigner samltoken.SignerProvider,
	attrSchemaRepo claimusecases.TenantAttributeSchemaRepo,
) {
	profileRepo := m.ProfileRepo
	if profileRepo == nil {
		profileRepo, _ = m.SPRepo.(ports.SamlIdentityProviderProfileRepository)
	}
	samlhttp.RegisterRoutes(g, samlhttp.Deps{
		Deps: deps, Authenticator: authenticator, ApplicationGate: applicationGate,
		SamlSPRepo: m.SPRepo, IDPProfileRepo: profileRepo, ReplayStore: m.ReplayStore,
		FederationSigner: federationSigner, UserRepo: userRepo, AttrSchemaRepo: attrSchemaRepo,
	})
}
