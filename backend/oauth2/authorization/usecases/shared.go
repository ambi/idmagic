package usecases

import sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"

type OAuthError = sharedusecases.OAuthError

var (
	NewOAuthError                = sharedusecases.NewOAuthError
	errorCode                    = sharedusecases.ErrorCode
	emit                         = sharedusecases.Emit
	generateOpaqueToken          = sharedusecases.GenerateOpaqueToken
	ParseAuthorizationDetails    = sharedusecases.ParseAuthorizationDetails
	ValidateAuthorizationDetails = sharedusecases.ValidateAuthorizationDetails
	ResolveResourceIndicator     = sharedusecases.ResolveResourceIndicator
)
