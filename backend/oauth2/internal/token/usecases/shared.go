package usecases

import sharedusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"

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
	RejectAgentClientWithoutKeyProof    = sharedusecases.RejectAgentClientWithoutKeyProof
	// 要求したクライアントの Agent を発行するトークンへ記録する (REQ-OAUTH2-012)。
	RequestingAgentID        = sharedusecases.RequestingAgentID
	IssuerForRequestingAgent = sharedusecases.IssuerForRequestingAgent
	IssuerWithAgentGuardrail = sharedusecases.IssuerWithAgentGuardrail
)

type AgentIssuanceDeps = sharedusecases.AgentIssuanceDeps

type AgentGuardrailDeps = sharedusecases.AgentGuardrailDeps
