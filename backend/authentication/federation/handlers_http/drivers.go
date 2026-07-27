package handlers_http

import (
	"context"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationports "github.com/ambi/idmagic/backend/authentication/federation/ports"
	oidcprotocol "github.com/ambi/idmagic/backend/authentication/federation/protocol_oidc"
	samlprotocol "github.com/ambi/idmagic/backend/authentication/federation/protocol_saml"
)

type OIDCDriver struct{ Client oidcprotocol.Client }

func (d OIDCDriver) Start(
	connection federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	callbackURL string,
	_ time.Time,
) (string, error) {
	return oidcprotocol.AuthorizationURL(connection, attempt, callbackURL)
}

func (d OIDCDriver) Complete(
	ctx context.Context,
	connection federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	response, callbackURL string,
	now time.Time,
) (federationdomain.NormalizedClaims, error) {
	return d.Client.ExchangeAndValidate(ctx, connection, attempt, response, callbackURL, now)
}

type SAMLDriver struct{ Replay federationports.ReplayStore }

func (d SAMLDriver) Start(
	connection federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	callbackURL string,
	now time.Time,
) (string, error) {
	return samlprotocol.BuildAuthnRequest(connection, attempt, callbackURL, now)
}

func (d SAMLDriver) Complete(
	ctx context.Context,
	connection federationdomain.IdentityProviderConnection,
	attempt federationdomain.FederatedLoginAttempt,
	response, callbackURL string,
	now time.Time,
) (federationdomain.NormalizedClaims, error) {
	return samlprotocol.ValidateResponse(ctx, connection, attempt, response, callbackURL, now, d.Replay)
}
