package db_postgres

import (
	clientpostgres "github.com/ambi/idmagic/backend/oauth2/client/db_postgres"
	consentpostgres "github.com/ambi/idmagic/backend/oauth2/consent/db_postgres"
	tokenpostgres "github.com/ambi/idmagic/backend/oauth2/token/db_postgres"
)

type (
	OAuth2ClientRepository = clientpostgres.OAuth2ClientRepository
	ConsentRepository      = consentpostgres.ConsentRepository
	RefreshTokenStore      = tokenpostgres.RefreshTokenStore
)
