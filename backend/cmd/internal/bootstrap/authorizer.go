package bootstrap

import (
	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	authorizationHTTP "github.com/ambi/idmagic/backend/shared/policy/authorization_http"
	authorizationLocal "github.com/ambi/idmagic/backend/shared/policy/authorization_local"
)

// AssembleAuthorizer builds the Authorizer selected by cfg.AuthZEN. cfg is
// assumed already validated by LoadSharedConfig (AUTHZEN is a closed enum;
// AUTHZEN_URL is required and must be an absolute URL when AUTHZEN=remote),
// so this function only constructs the adapter.
func AssembleAuthorizer(cfg SharedConfig) policyports.Authorizer {
	if cfg.AuthZEN == "remote" {
		return authorizationHTTP.NewRemote(cfg.AuthZENURL)
	}
	return authorizationLocal.Local{}
}
