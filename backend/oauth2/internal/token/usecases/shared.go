package usecases

import oauthusecases "github.com/ambi/idmagic/backend/oauth2/internal/usecases"

type OAuthError = oauthusecases.OAuthError

var (
	NewOAuthError                = oauthusecases.NewOAuthError
	emit                         = oauthusecases.Emit
	senderConstraintTag          = oauthusecases.SenderConstraintTag
	ParseAuthorizationDetails    = oauthusecases.ParseAuthorizationDetails
	ValidateAuthorizationDetails = oauthusecases.ValidateAuthorizationDetails
	LoadAuthorizationDetailTypes = oauthusecases.LoadAuthorizationDetailTypes
	ResolveResourceIndicator     = oauthusecases.ResolveResourceIndicator
	ResolveIssuableAgent         = oauthusecases.ResolveIssuableAgent
	// 承認を記録しない発行経路の区分ゲート (REQ-OAUTH2-050)。
	ResolveIssuableAgentWithoutApproval = oauthusecases.ResolveIssuableAgentWithoutApproval
	RejectAgentIssuanceWithoutApproval  = oauthusecases.RejectAgentIssuanceWithoutApproval
	RejectAgentClientWithoutKeyProof    = oauthusecases.RejectAgentClientWithoutKeyProof
	// 要求したクライアントの Agent を発行するトークンへ記録する (REQ-OAUTH2-012)。
	IssuerForRequestingAgent = oauthusecases.IssuerForRequestingAgent
	IssuerWithAgentGuardrail = oauthusecases.IssuerWithAgentGuardrail
)

type AgentIssuanceDeps = oauthusecases.AgentIssuanceDeps

type AgentGuardrailDeps = oauthusecases.AgentGuardrailDeps
