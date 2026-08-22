package usecases

import sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"

type OAuthError = sharedusecases.OAuthError

var (
	NewOAuthError                = sharedusecases.NewOAuthError
	emit                         = sharedusecases.Emit
	senderConstraintTag          = sharedusecases.SenderConstraintTag
	ParseAuthorizationDetails    = sharedusecases.ParseAuthorizationDetails
	ValidateAuthorizationDetails = sharedusecases.ValidateAuthorizationDetails
	LoadAuthorizationDetailTypes = sharedusecases.LoadAuthorizationDetailTypes
	ResolveResourceIndicator     = sharedusecases.ResolveResourceIndicator
	ResolveIssuableAgent         = sharedusecases.ResolveIssuableAgent
	// 承認を記録しない発行経路の区分ゲート (REQ-OAUTH2-050)。
	ResolveIssuableAgentWithoutApproval = sharedusecases.ResolveIssuableAgentWithoutApproval
	RejectAgentIssuanceWithoutApproval  = sharedusecases.RejectAgentIssuanceWithoutApproval
)

type AgentIssuanceDeps = sharedusecases.AgentIssuanceDeps
