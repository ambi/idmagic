package usecases

import (
	"context"
	"slices"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	agentGuardrailRatePolicyID = "agent_guardrail_issuance"

	agentGuardrailAllowed          = "allowed"
	agentGuardrailApproved         = "approved"
	agentGuardrailApprovalRequired = "approval_required"
	agentGuardrailDenied           = "denied"
)

// AgentGuardrailDeps は一つの要求の評価に要るものである。Policies が nil の配線では評価しない。
type AgentGuardrailDeps struct {
	Policies          ports.AgentGuardrailPolicyRepository
	Limiter           rlports.RateLimiter
	AgentRepo         agentports.AgentRepository
	UserRepo          userports.UserRepository
	Emit              func(spec.DomainEvent)
	ClientID          string
	GrantType         string
	RequestID         string
	RequestingAgentID string
	Now               time.Time
}

// IssuerWithAgentGuardrail は発行を関わる全 Agent のガードレールで評価する発行器を返す
// (REQ-OAUTH2-073)。グラントごとのユースケースではなく発行器で評価するのは、送信者制約と同じく、
// グラントを足すたびに書き写さずに済ませ、書き忘れた経路が上限の回避に使われることを防ぐためである。
func IssuerWithAgentGuardrail(issuer ports.TokenIssuer, deps AgentGuardrailDeps) ports.TokenIssuer {
	if deps.Policies == nil {
		return issuer
	}
	return &guardedIssuer{TokenIssuer: issuer, deps: deps}
}

type guardedIssuer struct {
	ports.TokenIssuer
	deps     AgentGuardrailDeps
	tenantID string
	// checked は Precheck を通った評価で、次の署名か Confirm が記録する。
	checked []agentGuardrailResult
	// lifetimeSeconds は要求に限って短くした有効期間である。0 は短くしていない。
	lifetimeSeconds int
	limited         int
}

type agentGuardrailResult struct {
	agentID  string
	outcome  string
	reasons  []string
	versions []string
}

func (g *guardedIssuer) AccessTokenTTLSeconds() int {
	if g.lifetimeSeconds > 0 {
		return g.lifetimeSeconds
	}
	return g.TokenIssuer.AccessTokenTTLSeconds()
}

func (g *guardedIssuer) SignAccessToken(ctx context.Context, in ports.AccessTokenInput) (string, string, error) {
	if err := g.Precheck(ctx, in); err != nil {
		return "", "", err
	}
	if g.limited > 0 && g.limited < g.TokenIssuer.AccessTokenTTLSeconds() {
		expiresAt := g.deps.Now.Unix() + int64(g.limited)
		if in.ExpiresAt == 0 || expiresAt < in.ExpiresAt {
			in.ExpiresAt = expiresAt
			g.lifetimeSeconds = g.limited
		}
	}
	token, jti, err := g.TokenIssuer.SignAccessToken(ctx, in)
	if err != nil {
		return "", "", err
	}
	g.Confirm(jti)
	return token, jti, nil
}

func (g *guardedIssuer) Confirm(tokenID string) {
	for _, result := range g.checked {
		g.record(result, tokenID)
	}
	g.checked = nil
}

// Precheck は発行を評価する。通った評価を次の署名が再利用するので、前判定と署名で二重に数えない。
func (g *guardedIssuer) Precheck(ctx context.Context, in ports.AccessTokenInput) error {
	g.tenantID = tenantports.TenantID(ctx)
	if g.checked != nil {
		return nil
	}
	agents := domain.MergeAgentChain([]string{g.deps.RequestingAgentID, in.AgentID}, in.AgentChain)
	if len(agents) == 0 {
		g.checked = []agentGuardrailResult{}
		return nil
	}
	policies, err := g.deps.Policies.ListEffective(ctx, tenantports.TenantID(ctx), agents)
	if err != nil {
		return g.reject(agentGuardrailResult{agentID: agents[0], reasons: []string{domain.AgentGuardrailReasonUnavailable}}, agentGuardrailDenied)
	}
	req := domain.AgentGuardrailRequest{Resources: in.Audiences, DelegationDepth: guardrailActDepth(in.Act)}
	for _, detail := range in.AuthorizationDetails {
		req.DetailTypes = append(req.DetailTypes, detail.Type)
	}

	evaluations := make([]agentGuardrailEvaluation, 0, len(agents))
	for _, agentID := range agents {
		layers := guardrailLayers(policies, agentID)
		if len(layers) > 0 {
			evaluations = append(evaluations, g.evaluateLayers(agentID, layers, req))
		}
	}
	// 計数は作用なので、作用のない判定をすべて通した後で消費する。
	results := make([]agentGuardrailResult, 0, len(evaluations))
	limited := 0
	for _, evaluation := range evaluations {
		if len(evaluation.denied) > 0 {
			return g.reject(evaluation.result(evaluation.denied), agentGuardrailDenied)
		}
	}
	for i := range evaluations {
		if err := g.requireWaiver(ctx, &evaluations[i], in); err != nil {
			return err
		}
	}
	for i := range evaluations {
		evaluation := &evaluations[i]
		if err := g.consumeRates(ctx, evaluation, in); err != nil {
			return err
		}
		outcome := agentGuardrailAllowed
		if evaluation.waived {
			outcome = agentGuardrailApproved
		}
		result := evaluation.result(evaluation.requiresApproval)
		result.outcome = outcome
		results = append(results, result)
		for _, layer := range evaluation.layers {
			lifetime := layer.TokenLifetimeSeconds(g.TokenIssuer.AccessTokenTTLSeconds(), g.deps.Now)
			if limited == 0 || lifetime < limited {
				limited = lifetime
			}
		}
	}
	g.checked, g.limited = results, limited
	return nil
}

type agentGuardrailEvaluation struct {
	agentID          string
	layers           []*domain.AgentGuardrailPolicy
	denied           []string
	requiresApproval []string
	waived           bool
}

func (e agentGuardrailEvaluation) result(reasons []string) agentGuardrailResult {
	versions := make([]string, 0, len(e.layers))
	for _, layer := range e.layers {
		versions = append(versions, layer.Version())
	}
	return agentGuardrailResult{agentID: e.agentID, reasons: slices.Clone(reasons), versions: versions}
}

func (g *guardedIssuer) evaluateLayers(agentID string, layers []*domain.AgentGuardrailPolicy, req domain.AgentGuardrailRequest) agentGuardrailEvaluation {
	evaluation := agentGuardrailEvaluation{agentID: agentID, layers: layers}
	for _, layer := range layers {
		for _, finding := range layer.Evaluate(req, g.deps.Now) {
			if finding.Action == domain.AgentGuardrailDeny {
				evaluation.denied = append(evaluation.denied, finding.Reason)
			} else {
				evaluation.requiresApproval = append(evaluation.requiresApproval, finding.Reason)
			}
		}
	}
	return evaluation
}

// requireWaiver は降格した超過を所有者の承認で免除する (REQ-OAUTH2-076)。
func (g *guardedIssuer) requireWaiver(ctx context.Context, evaluation *agentGuardrailEvaluation, in ports.AccessTokenInput) error {
	if len(evaluation.requiresApproval) == 0 || evaluation.waived {
		return nil
	}
	if in.ApprovalRequestID == "" {
		return g.reject(evaluation.result(evaluation.requiresApproval), agentGuardrailApprovalRequired)
	}
	approved, err := g.approvedByOwner(ctx, evaluation.agentID, in.Sub)
	if err != nil {
		return g.reject(evaluation.result([]string{domain.AgentGuardrailReasonUnavailable}), agentGuardrailDenied)
	}
	if !approved {
		return g.reject(evaluation.result(evaluation.requiresApproval), agentGuardrailDenied)
	}
	evaluation.waived = true
	return nil
}

func (g *guardedIssuer) approvedByOwner(ctx context.Context, agentID, sub string) (bool, error) {
	if g.deps.AgentRepo == nil || sub == "" {
		return false, nil
	}
	agent, err := g.deps.AgentRepo.FindByID(ctx, tenantports.TenantID(ctx), agentID)
	if err != nil || agent == nil || agent.OwnerUserID != sub {
		return false, err
	}
	return AgentOwnerIsActive(ctx, g.deps.UserRepo, agent)
}

func (g *guardedIssuer) consumeRates(ctx context.Context, evaluation *agentGuardrailEvaluation, in ports.AccessTokenInput) error {
	for _, layer := range evaluation.layers {
		rate := layer.IssuanceRate
		if rate == nil {
			continue
		}
		if g.deps.Limiter == nil {
			return g.reject(evaluation.result([]string{domain.AgentGuardrailReasonUnavailable}), agentGuardrailDenied)
		}
		allowed, err := g.deps.Limiter.AllowWithin(ctx, agentGuardrailRatePolicyID, guardrailRateKey(layer, evaluation.agentID),
			rlports.RateLimitPolicyConfig{MaxRequests: rate.MaxIssuances, WindowSeconds: rate.WindowSeconds}, g.deps.Now)
		if err != nil {
			return g.reject(evaluation.result([]string{domain.AgentGuardrailReasonUnavailable}), agentGuardrailDenied)
		}
		if allowed.Allowed {
			continue
		}
		reason := []string{domain.AgentGuardrailReasonIssuanceRate}
		if layer.OnViolation.IssuanceRate != domain.AgentGuardrailRequireApproval {
			return g.reject(evaluation.result(reason), agentGuardrailDenied)
		}
		evaluation.requiresApproval = append(evaluation.requiresApproval, reason...)
		if err := g.requireWaiver(ctx, evaluation, in); err != nil {
			return err
		}
	}
	return nil
}

// reject は評価を記録して拒否を返す。理由は監査にだけ残し、Agent が上限を探れないようにする。
func (g *guardedIssuer) reject(result agentGuardrailResult, outcome string) error {
	result.outcome = outcome
	g.record(result, "")
	if outcome == agentGuardrailApprovalRequired {
		return NewOAuthError("unauthorized_client", "human approval is required")
	}
	return NewOAuthError("access_denied", "The token request was denied.")
}

func (g *guardedIssuer) record(result agentGuardrailResult, tokenID string) {
	emit(g.deps.Emit, &domain.AgentGuardrailEvaluated{
		At: g.deps.Now, TenantID: g.tenantID, AgentID: result.agentID, ClientID: g.deps.ClientID, GrantType: g.deps.GrantType,
		Outcome: result.outcome, Reasons: result.reasons, PolicyVersions: result.versions, RequestID: g.deps.RequestID, TokenID: tokenID,
	})
}

func guardrailLayers(policies []*domain.AgentGuardrailPolicy, agentID string) []*domain.AgentGuardrailPolicy {
	var layers []*domain.AgentGuardrailPolicy
	for _, policy := range policies {
		if policy.AgentID == "" || policy.AgentID == agentID {
			layers = append(layers, policy)
		}
	}
	return layers
}

// guardrailRateKey は層ごとに計数を分ける。窓の異なる二層を一つの値へまとめられないからである。
func guardrailRateKey(layer *domain.AgentGuardrailPolicy, agentID string) string {
	if layer.AgentID == "" {
		return "default:" + agentID
	}
	return "agent:" + agentID
}

func guardrailActDepth(act map[string]any) int {
	depth := 0
	for act != nil {
		depth++
		nested, ok := act["act"].(map[string]any)
		if !ok {
			break
		}
		act = nested
	}
	return depth
}
