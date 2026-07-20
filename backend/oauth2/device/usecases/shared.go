package usecases

import sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"

type OAuthError = sharedusecases.OAuthError

var (
	NewOAuthError            = sharedusecases.NewOAuthError
	emit                     = sharedusecases.Emit
	ResolveResourceIndicator = sharedusecases.ResolveResourceIndicator
	senderConstraintTag      = sharedusecases.SenderConstraintTag
)
