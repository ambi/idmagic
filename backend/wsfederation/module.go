// Package wsfederation は WS-Federation モジュールの DI 組立を所有する。
package wsfederation

import (
	signinports "github.com/ambi/idmagic/backend/authentication/signin/ports"
	claimports "github.com/ambi/idmagic/backend/claimmapping/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	wsfedhttp "github.com/ambi/idmagic/backend/wsfederation/internal/handlers_http"
	wsfedusecases "github.com/ambi/idmagic/backend/wsfederation/internal/usecases"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"

	"github.com/labstack/echo/v5"
)

type Module struct {
	RPRepo wsfedports.WsFedRelyingPartyRepository
}

func (m Module) Register(g *echo.Group, deps support.Deps, authenticator *support.Authenticator,
	applicationGate wsfedhttp.ApplicationGate, sessions wsfedhttp.Sessions, secondFactors wsfedhttp.SecondFactorSteps,
	userRepo userports.UserRepository, federationSigner samltoken.SignerProvider,
	messageIDReplayStore wsfedports.MessageIDReplayStore, passwords signinports.PasswordAuthenticator,
	attrSchemaRepo claimports.TenantAttributeSchemaRepo,
) {
	wsfedhttp.RegisterRoutes(g, wsfedhttp.Deps{
		Deps: deps, Authenticator: authenticator, ApplicationGate: applicationGate, Sessions: sessions,
		SecondFactors: secondFactors, WsFedRPRepo: m.RPRepo,
		UserRepo: userRepo, FederationSigner: federationSigner, MessageIDReplayStore: messageIDReplayStore,
		Passwords: passwords, AttrSchemaRepo: attrSchemaRepo,
	})
}

// RelyingPartyAdministration は、Application のカタログが使う RP の管理操作を返す。
// RP の保存先がなければ nil を返す。
func (m Module) RelyingPartyAdministration() wsfedports.RelyingPartyAdministration {
	if m.RPRepo == nil {
		return nil
	}
	return wsfedusecases.RelyingPartyAdministration{RPRepo: m.RPRepo}
}
