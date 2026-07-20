package postgres

import (
	clientpostgres "github.com/ambi/idmagic/backend/oauth2/client/adapters/persistence/postgres"
	consentpostgres "github.com/ambi/idmagic/backend/oauth2/consent/adapters/persistence/postgres"
	tokenpostgres "github.com/ambi/idmagic/backend/oauth2/token/adapters/persistence/postgres"
)

type (
	OAuth2ClientRepository = clientpostgres.OAuth2ClientRepository
	ConsentRepository      = consentpostgres.ConsentRepository
	RefreshTokenStore      = tokenpostgres.RefreshTokenStore
)
