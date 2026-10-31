// ID-JAG の発行 (draft-ietf-oauth-identity-assertion-authz-grant-05 §4.3)。既存のトークン交換とは
// 入力、検証、出力を共有しない。act 必須やスコープの部分集合という交換の不変条件を緩めないためである。
package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	// TokenTypeIDJAGURN は ID-JAG を要求する requested_token_type である。
	TokenTypeIDJAGURN   = "urn:ietf:params:oauth:token-type:id-jag"
	tokenTypeIDTokenURN = "urn:ietf:params:oauth:token-type:id_token"
	idJagStageIssue     = "issue"
	idJagStageRedeem    = "redeem"
	idJagTokenTypeNA    = "N_A"
)

// ApplicationAccessCheck は、利用者が RP のクライアントの Application へ到達できるかを答える。
type ApplicationAccessCheck func(ctx context.Context, clientID string, assertion *ports.IDTokenAssertion) (bool, error)

type IssueIDJAGDeps struct {
	// Guard は発行を Agent のガードレールで評価する (REQ-OAUTH2-073)。
	Guard      ports.IssuanceGuard
	ClientRepo ports.OAuth2ClientRepository
	UserRepo   userports.UserRepository
	Policies   ports.CrossAppAccessPolicyRepository
	Verifier   ports.IDTokenAssertionVerifier
	Signer     ports.IDJAGSigner
	// ApplicationAccess は nil なら拒否する。
	ApplicationAccess ApplicationAccessCheck
	// UserEpochs は利用者の失効を引く (REQ-SHAREDSIGNALS-012)。
	UserEpochs ssports.UserRevocationEpochRepository
	AgentRepo  agentports.AgentRepository
	Consents   ports.ConsentRepository
	Emit       func(spec.DomainEvent)
}

type IssueIDJAGInput struct {
	ClientID             string
	SubjectToken         string
	SubjectTokenType     string
	HasActorToken        bool
	Audiences            []string
	Resources            []string
	Scope                string
	AuthorizationDetails []spec.AuthorizationDetail
	ProofJKT             string
	ProofX5TS256         string
}

// IssueIDJAG は REQ-OAUTH2-066〜068 に従って ID-JAG を発行する。
func IssueIDJAG(ctx context.Context, deps IssueIDJAGDeps, in IssueIDJAGInput, now time.Time) (*ExchangeTokenResult, error) {
	tenantID := tenantports.TenantID(ctx)
	reject := func(code, description string) error {
		emit(deps.Emit, &domain.IdJagRejected{At: now, TenantID: tenantID, ClientID: in.ClientID, Stage: idJagStageIssue, Reason: code})
		return NewOAuthError(code, description)
	}

	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil || client.ClientType != spec.ClientConfidential {
		return nil, reject("invalid_client", "A confidential client is required.")
	}
	if in.HasActorToken {
		return nil, reject("invalid_request", "actor_token is not supported for this requested_token_type.")
	}
	if in.SubjectTokenType != tokenTypeIDTokenURN {
		return nil, reject("invalid_request", "unsupported subject_token_type")
	}
	audiences := nonEmpty(in.Audiences)
	if len(audiences) != 1 {
		return nil, reject("invalid_request", "exactly one audience is required")
	}
	audience := audiences[0]

	assertion, err := deps.Verifier.VerifyIDTokenAssertion(ctx, in.SubjectToken)
	if err != nil || len(assertion.Audiences) != 1 || assertion.Audiences[0] != in.ClientID {
		return nil, reject("invalid_grant", "The subject_token is invalid.")
	}

	policies, err := deps.Policies.ListIssuePolicies(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	known := false
	var policy *domain.CrossAppAccessIssuePolicy
	for _, candidate := range policies {
		if candidate.Audience != audience {
			continue
		}
		known = true
		if candidate.ClientID == in.ClientID {
			policy = candidate
		}
	}
	if !known {
		return nil, reject("invalid_target", "The audience is not a trusted authorization server.")
	}
	if policy == nil {
		return nil, reject("access_denied", "The client is not authorized for the audience.")
	}

	resources := nonEmpty(in.Resources)
	if len(resources) == 0 {
		return nil, reject("invalid_target", "resource is required")
	}
	for _, resource := range resources {
		if !slices.Contains(policy.Resources, resource) {
			return nil, reject("invalid_target", "The resource is not governed by the audience.")
		}
	}
	scopes := domain.CrossAppAccessGrantedScopes(strings.Fields(in.Scope), policy.Scopes)
	if len(scopes) == 0 {
		return nil, reject("access_denied", "No requested scope is permitted.")
	}

	user, err := deps.UserRepo.FindBySub(ctx, assertion.Subject)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenantID || !user.IsActive() {
		return nil, reject("access_denied", "The access is not permitted.")
	}
	if deps.UserEpochs != nil {
		epoch, err := deps.UserEpochs.FindByUser(ctx, tenantID, assertion.Subject)
		if err != nil {
			return nil, err
		}
		if epoch != nil && epoch.Supersedes(assertion.IssuedAt) {
			return nil, reject("invalid_grant", "The subject_token is invalid.")
		}
	}
	if deps.ApplicationAccess == nil {
		return nil, reject("access_denied", "The access is not permitted.")
	}
	permitted, err := deps.ApplicationAccess(ctx, policy.RelyingPartyClientID, assertion)
	if err != nil {
		return nil, err
	}
	if !permitted {
		return nil, reject("access_denied", "The access is not permitted.")
	}

	if err := checkIDJAGAgent(ctx, idJagAgentDeps{
		ClientRepo: deps.ClientRepo, AgentRepo: deps.AgentRepo, Consents: deps.Consents, Emit: deps.Emit,
	}, in.ClientID, assertion.Subject, nil, now); err != nil {
		if oe, ok := errors.AsType[*OAuthError](err); ok {
			return nil, reject(oe.Code, oe.Description)
		}
		return nil, err
	}

	details := filterAuthorizationDetailsByType(in.AuthorizationDetails, policy.AuthorizationDetailTypes)
	if deps.Guard != nil {
		if err := deps.Guard.Precheck(ctx, ports.AccessTokenInput{Sub: assertion.Subject, Scopes: scopes, Audiences: resources, AuthorizationDetails: details}); err != nil {
			if oe, ok := errors.AsType[*OAuthError](err); ok {
				return nil, reject(oe.Code, oe.Description)
			}
			return nil, err
		}
	}
	input := ports.IDJAGInput{
		Subject: assertion.Subject, Audience: policy.Audience, ClientID: policy.TargetClientID,
		Resources: resources, Scopes: scopes, AuthorizationDetails: details,
		AuthTime: assertion.AuthTime, AMR: assertion.AMR, ACR: assertion.ACR,
		SenderConstraint: idJagBinding(in.ProofJKT, in.ProofX5TS256),
	}
	if policy.ReleaseEmail && user.Email != nil && user.EmailVerified {
		input.Email = *user.Email
	}
	token, jti, err := deps.Signer.SignIDJAG(ctx, input)
	if err != nil {
		return nil, err
	}
	if deps.Guard != nil {
		deps.Guard.Confirm(jti)
	}
	emit(deps.Emit, &domain.IdJagIssued{
		At: now, TenantID: tenantID, ClientID: in.ClientID, UserID: assertion.Subject,
		Audience: policy.Audience, Resources: resources, Scopes: scopes, JTI: jti,
	})
	return &ExchangeTokenResult{
		AccessToken: token, IssuedTokenType: TokenTypeIDJAGURN, TokenType: idJagTokenTypeNA,
		ExpiresIn: deps.Signer.IDJAGTTLSeconds(), Scope: strings.Join(scopes, " "), AuthorizationDetails: details,
	}, nil
}

// idJagBinding は REQ-OAUTH2-068 の表である。両方の証明があれば DPoP を選ぶ。
func idJagBinding(jkt, x5t string) *domain.SenderConstraint {
	switch {
	case jkt != "":
		return &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: jkt}
	case x5t != "":
		return &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: x5t}
	}
	return nil
}

func filterAuthorizationDetailsByType(details []spec.AuthorizationDetail, allowed []string) []spec.AuthorizationDetail {
	var out []spec.AuthorizationDetail
	for _, detail := range details {
		if slices.Contains(allowed, detail.Type) {
			out = append(out, detail)
		}
	}
	return out
}

type idJagAgentDeps struct {
	ClientRepo ports.OAuth2ClientRepository
	AgentRepo  agentports.AgentRepository
	Consents   ports.ConsentRepository
	Emit       func(spec.DomainEvent)
}

// checkIDJAGAgent は要求したクライアントの Agent を、交換と同じ規則で確かめる
// (REQ-OAUTH2-050、REQ-OAUTH2-060)。発行のホップでは scopes を渡さない。スコープの意味は
// リソースの認可サーバーのものなので、付与の有無だけを確かめる。
func checkIDJAGAgent(ctx context.Context, deps idJagAgentDeps, clientID, sub string, scopes []string, now time.Time) error {
	if deps.AgentRepo == nil {
		return nil
	}
	tenantID := tenantports.TenantID(ctx)
	acting, rejection := rejectSupervisedAgents(ctx, ExchangeTokenDeps{
		AgentRepo: deps.AgentRepo, Emit: deps.Emit,
	}, ExchangeTokenInput{ClientID: clientID}, nil, nil, now)
	if rejection != nil {
		return rejection
	}
	if acting == "" {
		return nil
	}
	granted, err := delegationsPermit(ctx, deps.Consents, tenantID, sub, []string{acting}, scopes, now)
	if err != nil {
		return err
	}
	if !granted {
		return NewOAuthError("access_denied", "The user has not granted delegation to the agent.")
	}
	return deps.Consents.TouchAgentDelegation(ctx, tenantID, sub, acting, now)
}
