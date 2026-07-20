package bootstrap

import (
	"errors"
	"os"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	authorizationHTTP "github.com/ambi/idmagic/backend/shared/policy/authorization_http"
	authorizationLocal "github.com/ambi/idmagic/backend/shared/policy/authorization_local"
)

func AssembleAuthorizer() (oauthports.Authorizer, error) {
	switch EnvDefault("AUTHZEN", "local") {
	case "local":
		return authorizationLocal.Local{}, nil
	case "remote":
		endpoint := os.Getenv("AUTHZEN_URL")
		if endpoint == "" {
			return nil, errors.New("AUTHZEN=remote requires AUTHZEN_URL")
		}
		return authorizationHTTP.NewRemote(endpoint), nil
	default:
		return nil, errors.New("AUTHZEN must be local or remote")
	}
}
