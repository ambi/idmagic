package usecases

import oauthusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"

type OAuthError = oauthusecases.OAuthError

var (
	NewOAuthError            = oauthusecases.NewOAuthError
	emit                     = oauthusecases.Emit
	ResolveResourceIndicator = oauthusecases.ResolveResourceIndicator
	senderConstraintTag      = oauthusecases.SenderConstraintTag
)
