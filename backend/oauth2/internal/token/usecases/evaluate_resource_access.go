// /resource-access/evaluate の中核 (REQ-OAUTH2-062、REQ-OAUTH2-063)。
package usecases

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrResourceServerNotRegistered は、認証したクライアントに束縛した Active の登録がないことを表す。
var ErrResourceServerNotRegistered = errors.New("no active resource server is bound to the client")

// ErrResourceAccessNotConfigured は、必須の依存を配線していない組み立てを表す。
var ErrResourceAccessNotConfigured = errors.New("resource access evaluation is not configured")

// ResourceAccessDeps は評価の境界である。Tokens は失効の判定まで済ませた結果を返す。
type ResourceAccessDeps struct {
	Servers   ports.McpResourceServerRepository
	Tokens    ports.TokenIntrospector
	Clients   ports.OAuth2ClientRepository
	Relations policyports.RelationChecker
	Replay    ports.DpopReplayStore
	Emit      func(spec.DomainEvent)
}

// ResourceAccessRequest は 1 回の評価の入力である。ClientID は認証済みの値で、主体と行為者は
// 受け取らない。
type ResourceAccessRequest struct {
	TenantID              string
	ClientID              string
	Token                 string
	ResourceType          string
	ResourceID            string
	Relation              string
	DPoPProof             string
	HTM                   string
	HTU                   string
	CertificateThumbprint string
}

// ResourceAccessDecision は評価の結果である。不許可ではトークンのクレームを持たない。
type ResourceAccessDecision struct {
	Permitted            bool
	Reasons              []string
	Scope                string
	DelegationMode       domain.DelegationMode
	AuthorizationDetails []spec.AuthorizationDetail
}

// EvaluateResourceAccess は登録、規則、トークン、提示の証明、関係の判定の論理積で許可する。
// error は評価不能を表し、呼び出し側はこれを許可へ退避してはならない。
func EvaluateResourceAccess(ctx context.Context, deps ResourceAccessDeps, in ResourceAccessRequest, now time.Time) (ResourceAccessDecision, error) {
	if deps.Servers == nil || deps.Tokens == nil || deps.Relations == nil {
		return ResourceAccessDecision{}, ErrResourceAccessNotConfigured
	}
	server, err := deps.Servers.FindByClientID(ctx, in.TenantID, in.ClientID)
	if err != nil {
		return ResourceAccessDecision{}, err
	}
	if server == nil || !server.IsActive() {
		return ResourceAccessDecision{}, ErrResourceServerNotRegistered
	}
	decision, tokenID, err := evaluateForServer(ctx, deps, server, in, now)
	if err != nil {
		return ResourceAccessDecision{}, err
	}
	if deps.Emit != nil {
		deps.Emit(&domain.ResourceAccessEvaluated{
			At: now, TenantID: in.TenantID, RSClientID: in.ClientID, ResourceServerID: server.ID,
			TokenID: tokenID, ResourceType: in.ResourceType, Relation: in.Relation,
			Permitted: decision.Permitted, Reasons: decision.Reasons,
		})
	}
	return decision, nil
}

func evaluateForServer(
	ctx context.Context, deps ResourceAccessDeps, server *domain.McpResourceServer, in ResourceAccessRequest, now time.Time,
) (ResourceAccessDecision, string, error) {
	rule, ok := server.RuleFor(in.ResourceType, in.Relation)
	if !ok {
		return denyResourceAccess(domain.ReasonOperationNotRegistered), "", nil
	}
	token, err := deps.Tokens.IntrospectAccessToken(ctx, in.Token)
	if err != nil {
		return ResourceAccessDecision{}, "", err
	}
	if token == nil || !token.Active || !slices.Contains(token.Aud, server.Resource) {
		return denyResourceAccess(domain.ReasonTokenRejected), "", nil
	}
	proved, err := senderProofHolds(ctx, deps, server, token, in, now)
	if err != nil {
		return ResourceAccessDecision{}, token.JTI, err
	}
	if !proved {
		return denyResourceAccess(domain.ReasonSenderProofInvalid), token.JTI, nil
	}
	if !rule.PermitsDetails(token.AuthorizationDetails, server.Resource, in.ResourceID) {
		return denyResourceAccess(domain.ReasonAuthorizationDetailMismatch), token.JTI, nil
	}
	subject, actors, resolved, err := principalsFromToken(ctx, deps.Clients, in.TenantID, token)
	if err != nil {
		return ResourceAccessDecision{}, token.JTI, err
	}
	if !resolved {
		return denyResourceAccess(domain.ReasonPrincipalUnresolved), token.JTI, nil
	}
	relation, err := deps.Relations.CheckRelation(ctx, policyports.RelationCheck{
		TenantID: in.TenantID, ResourceType: in.ResourceType, ResourceID: in.ResourceID, Relation: in.Relation,
		Subject: subject, Actors: actors,
		GrantedScopes: strings.Fields(token.Scope), RequiredScopes: rule.Scopes,
	})
	if err != nil {
		return ResourceAccessDecision{}, token.JTI, err
	}
	if !relation.Permitted {
		return ResourceAccessDecision{Reasons: relation.Reasons}, token.JTI, nil
	}
	return ResourceAccessDecision{
		Permitted: true,
		Scope:     token.Scope,
		DelegationMode: domain.DeriveDelegationMode(domain.DelegationSubject{
			Sub: token.Sub, ClientID: token.ClientID, PrincipalType: token.PrincipalType, Act: token.Act,
		}),
		AuthorizationDetails: token.AuthorizationDetails,
	}, token.JTI, nil
}

func denyResourceAccess(reason string) ResourceAccessDecision {
	return ResourceAccessDecision{Reasons: []string{reason}}
}

// senderProofHolds は送信者制約に応じた提示の証明を確かめる。制約のないトークンでは、送られた
// 証明を検証せず、jti も記録しない。リプレイ記録へ届かないときだけ error を返す。
func senderProofHolds(
	ctx context.Context, deps ResourceAccessDeps, server *domain.McpResourceServer,
	token *ports.IntrospectionResult, in ResourceAccessRequest, now time.Time,
) (bool, error) {
	constraint := token.SenderConstraint
	if constraint == nil {
		return true, nil
	}
	switch constraint.Type {
	case spec.SenderConstraintDPoP:
		if in.DPoPProof == "" || !htuWithinResource(in.HTU, server.Resource) {
			return false, nil
		}
		if deps.Replay == nil {
			return false, ErrResourceAccessNotConfigured
		}
		verified, err := tokens_jose.VerifyDPoPForResource(ctx, in.DPoPProof, in.HTM, in.HTU, in.Token, deps.Replay, now)
		if errors.Is(err, tokens_jose.ErrDPoPReplayStoreUnavailable) {
			return false, err
		}
		if err != nil || verified == nil {
			return false, nil //nolint:nilerr // 証明の不正は評価不能ではなく不許可である。
		}
		return subtle.ConstantTimeCompare([]byte(verified.JKT), []byte(constraint.JKT)) == 1, nil
	case spec.SenderConstraintMTLS:
		return in.CertificateThumbprint != "" &&
			subtle.ConstantTimeCompare([]byte(in.CertificateThumbprint), []byte(constraint.X5TS256)) == 1, nil
	default:
		return false, nil
	}
}

// htuWithinResource は、証明の対象が登録の resource と同じオリジンの配下にあることを確かめる。
// 文字列の前方一致では https://mcp.example.evil/ のような別のホストを通してしまう。
func htuWithinResource(htu, resource string) bool {
	target, err := url.Parse(htu)
	if err != nil {
		return false
	}
	base, err := url.Parse(resource)
	if err != nil || target.Scheme != base.Scheme || !strings.EqualFold(target.Host, base.Host) {
		return false
	}
	prefix := strings.TrimSuffix(base.Path, "/")
	return target.Path == prefix || strings.HasPrefix(target.Path, prefix+"/")
}

// principalsFromToken は主体と行為者を検証済みのトークンだけから導く。Agent へ解決できない act を
// 持つトークンは、代行の連鎖を確かめられないので解決できないものとして扱う。
func principalsFromToken(
	ctx context.Context, clients ports.OAuth2ClientRepository, tenantID string, token *ports.IntrospectionResult,
) (policyports.PrincipalRef, []policyports.PrincipalRef, bool, error) {
	agents := domain.MergeAgentChain([]string{token.AgentID}, token.AgentChain)
	var subject policyports.PrincipalRef
	if token.PrincipalType == domain.PrincipalTypeAgent && token.AgentID != "" {
		subject = policyports.PrincipalRef{Type: domain.PrincipalTypeAgent, ID: token.AgentID}
	} else {
		user, err := isUserSubject(ctx, clients, tenantID, token.Sub)
		if err != nil {
			return policyports.PrincipalRef{}, nil, false, err
		}
		if !user {
			return policyports.PrincipalRef{}, nil, false, nil
		}
		subject = policyports.PrincipalRef{Type: "user", ID: token.Sub}
	}
	actors := make([]policyports.PrincipalRef, 0, len(agents))
	for _, agentID := range agents {
		if subject.Type == domain.PrincipalTypeAgent && agentID == subject.ID {
			continue
		}
		actors = append(actors, policyports.PrincipalRef{Type: domain.PrincipalTypeAgent, ID: agentID})
	}
	if len(token.Act) > 0 && len(actors) == 0 {
		return policyports.PrincipalRef{}, nil, false, nil
	}
	return subject, actors, true, nil
}
