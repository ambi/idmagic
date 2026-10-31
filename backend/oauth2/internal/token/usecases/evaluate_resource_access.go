// /resource-access/evaluate の中核 (REQ-OAUTH2-062、REQ-OAUTH2-063)。
package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	policyports "github.com/ambi/idmagic/backend/accesscontrol/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	approvalports "github.com/ambi/idmagic/backend/oauth2/internal/approval/ports"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// ErrResourceServerNotRegistered は、認証したクライアントに束縛した Active の登録がないことを表す。
var ErrResourceServerNotRegistered = errors.New("no active resource server is bound to the client")

// ErrResourceAccessNotConfigured は、必須の依存を配線していない組み立てを表す。
var ErrResourceAccessNotConfigured = errors.New("resource access evaluation is not configured")

// ErrOperationKeyRequired は、承認した操作の評価に operation_key がないことを表す。
var ErrOperationKeyRequired = errors.New("operation_key is required")

// ResourceAccessDeps は評価の境界である。Tokens は失効の判定まで済ませた結果を返す。
type ResourceAccessDeps struct {
	Servers   ports.McpResourceServerRepository
	Tokens    ports.TokenIntrospector
	Clients   ports.OAuth2ClientRepository
	Relations policyports.RelationChecker
	Replay    ports.DpopReplayStore
	// Approvals は承認した操作の照合と使用権に使う。該当の規則を未配線で評価しない。
	Approvals approvalports.ApprovalRequestStore
	// Limiter は規則の上限を数える。上限を持つ規則を未配線で評価しない。
	Limiter rlports.RateLimiter
	// Grants は交換の許可を保存する。交換を求める評価を未配線で許可しない。
	Grants ports.TokenExchangeGrantStore
	Emit   func(spec.DomainEvent)
}

// resourceAccessLimitPolicyID は規則の上限の回数の名前空間で、入口の resource_access の予算と分ける。
const resourceAccessLimitPolicyID = "resource_access_rule_limit"

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
	OperationKey          string
	OperationParameters   string
	// Attributes はリソースサーバーが申告する条件の入力で、規則が条件を持つときだけ読む。
	Attributes string
	// TokenExchangeResource は許可の後に交換する宛先である (REQ-OAUTH2-079)。
	TokenExchangeResource string
	// RequestID は応答の X-Request-ID と同じ値で、監査イベントの相関に使う。
	RequestID string
}

// ApprovedOperationUse は、許可した承認済みの操作をリソースサーバーの結果と結ぶ相関である。
type ApprovedOperationUse struct {
	ApprovalRequestID string
	OperationDigest   string
	Replayed          bool
}

// ResourceAccessDecision は評価の結果である。不許可ではトークンのクレームを持たない。
type ResourceAccessDecision struct {
	Permitted            bool
	Reasons              []string
	Scope                string
	DelegationMode       domain.DelegationMode
	AuthorizationDetails []spec.AuthorizationDetail
	ApprovedOperation    *ApprovedOperationUse
	// TokenExchangeGrant は宛先への一回限りの交換の許可で、ログと監査へ出さない。
	TokenExchangeGrant string
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
	decision, audit, err := evaluateForServer(ctx, deps, server, in, now)
	if err != nil {
		return ResourceAccessDecision{}, err
	}
	if deps.Emit != nil {
		event := &domain.ResourceAccessEvaluated{
			At: now, TenantID: in.TenantID, RSClientID: in.ClientID, ResourceServerID: server.ID,
			TokenID: audit.tokenID, UserID: audit.userID, AgentID: audit.agentID, ActorChain: audit.actorChain, RequestID: in.RequestID,
			ResourceType: in.ResourceType, Relation: in.Relation,
			Permitted: decision.Permitted, Reasons: decision.Reasons,
		}
		if use := decision.ApprovedOperation; use != nil {
			event.ApprovalRequestID, event.OperationDigest, event.Replayed = use.ApprovalRequestID, use.OperationDigest, use.Replayed
		}
		event.ConditionVersion, event.ConditionOutcome, event.DeclaredAttributesDigest = audit.conditionVersion, string(audit.conditionOutcome), audit.attributesDigest
		if exceeded := audit.exceededLimit; exceeded != nil {
			event.ExceededLimitBy, event.ExceededLimitWindowSeconds = exceeded.By, exceeded.WindowSeconds
		}
		deps.Emit(event)
	}
	return decision, nil
}

// evaluationAudit は判断を記録するための相関である。主体は、トークンと提示の証明が成り立つときだけ持つ。
// 成り立たないトークンの持ち主を記録すると、盗んだトークンの試行が本人の操作に見える (REQ-OAUTH2-062)。
type evaluationAudit struct {
	tokenID    string
	userID     string
	agentID    string
	actorChain []string
	// 以下は評価した条件の相関。関係の判定まで届いた評価だけが持つ。
	conditionVersion string
	conditionOutcome domain.ConditionOutcome
	attributesDigest string
	// exceededLimit は不許可の原因になった上限である。
	exceededLimit *domain.ResourceAccessLimit
}

func newEvaluationAudit(tokenID string, subject policyports.PrincipalRef, actors []policyports.PrincipalRef) evaluationAudit {
	audit := evaluationAudit{tokenID: tokenID}
	if subject.Type == domain.PrincipalTypeAgent {
		audit.agentID = subject.ID
	} else {
		audit.userID = subject.ID
	}
	for _, actor := range actors {
		audit.actorChain = append(audit.actorChain, actor.ID)
	}
	if audit.agentID == "" && len(audit.actorChain) > 0 {
		audit.agentID = audit.actorChain[0]
	}
	return audit
}

func evaluateForServer(
	ctx context.Context, deps ResourceAccessDeps, server *domain.McpResourceServer, in ResourceAccessRequest, now time.Time,
) (ResourceAccessDecision, evaluationAudit, error) {
	rule, registered := server.RuleFor(in.ResourceType, in.Relation)
	if registered && rule.RequiresApprovedOperation && in.OperationKey == "" {
		return ResourceAccessDecision{}, evaluationAudit{}, ErrOperationKeyRequired
	}
	// 規則にない操作も試行者を記録するためにトークンを確かめるが、理由は語り分けない。
	reasonFor := func(reason string) string {
		if !registered {
			return domain.ReasonOperationNotRegistered
		}
		return reason
	}
	token, err := deps.Tokens.IntrospectAccessToken(ctx, in.Token)
	if err != nil {
		return ResourceAccessDecision{}, evaluationAudit{}, err
	}
	if token == nil || !token.Active || !slices.Contains(token.Aud, server.Resource) {
		return denyResourceAccess(reasonFor(domain.ReasonTokenRejected)), evaluationAudit{}, nil
	}
	audit := evaluationAudit{tokenID: token.JTI}
	proved, err := senderProofHolds(ctx, deps, server, token, in, now)
	if err != nil {
		return ResourceAccessDecision{}, audit, err
	}
	if !proved {
		return denyResourceAccess(reasonFor(domain.ReasonSenderProofInvalid)), audit, nil
	}
	subject, actors, resolved, err := principalsFromToken(ctx, deps, in.TenantID, token)
	if err != nil {
		return ResourceAccessDecision{}, audit, err
	}
	if resolved {
		audit = newEvaluationAudit(token.JTI, subject, actors)
	}
	if !registered {
		return denyResourceAccess(domain.ReasonOperationNotRegistered), audit, nil
	}
	if !rule.PermitsDetails(token.AuthorizationDetails, server.Resource, in.ResourceID) {
		return denyResourceAccess(domain.ReasonAuthorizationDetailMismatch), audit, nil
	}
	if !resolved {
		return denyResourceAccess(domain.ReasonPrincipalUnresolved), audit, nil
	}
	relation, err := deps.Relations.CheckRelation(ctx, policyports.RelationCheck{
		TenantID: in.TenantID, ResourceType: in.ResourceType, ResourceID: in.ResourceID, Relation: in.Relation,
		Subject: subject, Actors: actors,
		GrantedScopes: strings.Fields(token.Scope), RequiredScopes: rule.Scopes,
	})
	if err != nil {
		return ResourceAccessDecision{}, audit, err
	}
	if !relation.Permitted {
		return ResourceAccessDecision{Reasons: relation.Reasons}, audit, nil
	}
	approvalDetailType := ""
	if rule.RequiresApprovedOperation {
		approvalDetailType = rule.AuthorizationDetailType
	}
	if rule.Condition != nil {
		attributes, digest := declaredAttributes(in.Attributes)
		outcome := domain.ConditionIndeterminate
		if attributes != nil {
			outcome = rule.Condition.Evaluate(conditionInput(token, in, attributes, now))
		}
		audit.conditionVersion, audit.conditionOutcome, audit.attributesDigest = rule.Condition.Version(), outcome, digest
		switch outcome {
		case domain.ConditionDeny:
			return denyResourceAccess(domain.ReasonConditionDenied), audit, nil
		case domain.ConditionIndeterminate:
			return denyResourceAccess(domain.ReasonConditionIndeterminate), audit, nil
		case domain.ConditionRequireApproval:
			approvalDetailType = rule.Condition.ApprovalDetailType
			detail := domain.ResourceAccessRule{AuthorizationDetailType: approvalDetailType, Relation: in.Relation}
			if token.ApprovalRequestID == "" || !detail.PermitsDetails(token.AuthorizationDetails, server.Resource, in.ResourceID) {
				return denyResourceAccess(domain.ReasonApprovalRequired), audit, nil
			}
			// 規則だけでは鍵が要るか決まらないので、承認の経路へ進むときに確かめる。
			if in.OperationKey == "" {
				return ResourceAccessDecision{}, audit, ErrOperationKeyRequired
			}
		}
	}
	// 承認の使用権と上限の回数を減らす前に断る。交換の派生は承認を引き継がないので、承認の操作には出さない。
	if in.TokenExchangeResource != "" && (approvalDetailType != "" || !rule.PermitsTokenExchange(in.TokenExchangeResource) || token.JTI == "") {
		return denyResourceAccess(domain.ReasonTokenExchangeNotPermitted), audit, nil
	}
	var approval *matchedApproval
	if approvalDetailType != "" {
		matched, reason, err := matchApprovedOperation(ctx, deps, server, approvalDetailType, token, in, now)
		if err != nil {
			return ResourceAccessDecision{}, audit, err
		}
		if reason != "" {
			return denyResourceAccess(reason), audit, nil
		}
		approval = matched
	}
	// 上限は照合の後に数え、不一致の要求に予算を減らさせない。
	limit, exceeded, err := enforceResourceAccessLimits(ctx, deps, server, rule, audit, in, now)
	if err != nil {
		return ResourceAccessDecision{}, audit, err
	}
	if exceeded {
		audit.exceededLimit = &limit
		return denyResourceAccess(domain.ReasonLimitExceeded), audit, nil
	}
	var approved *ApprovedOperationUse
	if approval != nil {
		// 使用権は最後に得る。ほかの理由で不許可の要求に承認を消費させない。
		use, reason, err := claimApprovedOperation(ctx, deps, approval, now)
		if err != nil {
			return ResourceAccessDecision{}, audit, err
		}
		if reason != "" {
			return denyResourceAccess(reason), audit, nil
		}
		approved = use
	}
	grant, err := issueTokenExchangeGrant(ctx, deps, in, token.JTI, now)
	if err != nil {
		return ResourceAccessDecision{}, audit, err
	}
	return ResourceAccessDecision{
		Permitted:          true,
		ApprovedOperation:  approved,
		TokenExchangeGrant: grant,
		Scope:              token.Scope,
		DelegationMode: domain.DeriveDelegationMode(domain.DelegationSubject{
			Sub: token.Sub, ClientID: token.ClientID, PrincipalType: token.PrincipalType, Act: token.Act,
		}),
		AuthorizationDetails: token.AuthorizationDetails,
	}, audit, nil
}

// issueTokenExchangeGrant は許可の判断の後に許可を保存する。保存できなければ評価不能にし、値のない許可を返さない。
func issueTokenExchangeGrant(ctx context.Context, deps ResourceAccessDeps, in ResourceAccessRequest, subjectTokenID string, now time.Time) (string, error) {
	if in.TokenExchangeResource == "" {
		return "", nil
	}
	if deps.Grants == nil {
		return "", ErrResourceAccessNotConfigured
	}
	grant := rand.Text()
	if err := deps.Grants.Save(ctx, domain.HashTokenExchangeGrant(grant), domain.TokenExchangeGrant{
		ClientID: in.ClientID, SubjectTokenID: subjectTokenID, Resource: in.TokenExchangeResource,
		ExpiresAt: now.Add(domain.TokenExchangeGrantTTL),
	}); err != nil {
		return "", err
	}
	return grant, nil
}

// enforceResourceAccessLimits は規則の上限を順に数え、超えた上限を返す。超えた上限より後は数えない。
func enforceResourceAccessLimits(
	ctx context.Context, deps ResourceAccessDeps, server *domain.McpResourceServer, rule domain.ResourceAccessRule,
	audit evaluationAudit, in ResourceAccessRequest, now time.Time,
) (domain.ResourceAccessLimit, bool, error) {
	if len(rule.Limits) == 0 {
		return domain.ResourceAccessLimit{}, false, nil
	}
	if deps.Limiter == nil {
		return domain.ResourceAccessLimit{}, false, ErrResourceAccessNotConfigured
	}
	subject := domain.ResourceAccessLimitSubject{AgentID: audit.agentID, UserID: audit.userID, ResourceID: in.ResourceID}
	for _, limit := range rule.OrderedLimits() {
		key, applies := resourceAccessLimitKey(server, rule, limit, subject, in.TenantID)
		if !applies {
			continue
		}
		result, err := deps.Limiter.AllowWithin(ctx, resourceAccessLimitPolicyID, key,
			rlports.RateLimitPolicyConfig{MaxRequests: limit.MaxRequests, WindowSeconds: limit.WindowSeconds}, now)
		if err != nil {
			return domain.ResourceAccessLimit{}, false, err
		}
		if !result.Allowed {
			return limit, true, nil
		}
	}
	return domain.ResourceAccessLimit{}, false, nil
}

// resourceAccessLimitKey は回数の鍵を作る。識別子を含むので、応答とログへ出さない。
func resourceAccessLimitKey(
	server *domain.McpResourceServer, rule domain.ResourceAccessRule, limit domain.ResourceAccessLimit,
	subject domain.ResourceAccessLimitSubject, tenantID string,
) (string, bool) {
	bucket, applies := limit.Bucket(subject)
	if !applies {
		return "", false
	}
	return fmt.Sprintf("%q:%q:%q:%q:%s", tenantID, server.ID, rule.ResourceType, rule.Relation, bucket), true
}

// matchedApproval は照合を通った承認で、使用権の記録に渡す。
type matchedApproval struct {
	requestID string
	digest    string
	keyHash   string
}

// matchApprovedOperation は実際の操作を承認と照合する。読むだけで、使用権を記録しない (REQ-OAUTH2-065)。
func matchApprovedOperation(
	ctx context.Context, deps ResourceAccessDeps, server *domain.McpResourceServer, detailType string,
	token *ports.IntrospectionResult, in ResourceAccessRequest, now time.Time,
) (*matchedApproval, string, error) {
	if deps.Approvals == nil {
		return nil, "", ErrResourceAccessNotConfigured
	}
	if token.ApprovalRequestID == "" {
		return nil, domain.ReasonApprovalRequired, nil
	}
	rec, err := deps.Approvals.FindByID(ctx, token.ApprovalRequestID)
	if err != nil {
		return nil, "", err
	}
	if rec == nil || rec.UserID != token.Sub || rec.ClientID != token.ClientID {
		return nil, domain.ReasonApprovalExpired, nil
	}
	digest, err := approvaldomain.CanonicalizeOperationParameters(in.OperationParameters)
	if err != nil {
		return nil, domain.ReasonApprovedOperationMismatch, nil //nolint:nilerr // 比較できない引数は評価不能ではなく不一致である。
	}
	switch approvaldomain.MatchApprovedOperation(rec, approvaldomain.OperationBinding{
		DetailType: detailType, Location: server.Resource, Action: in.Relation, Identifier: in.ResourceID,
	}, digest, now) {
	case approvaldomain.ApprovedOperationExpired:
		return nil, domain.ReasonApprovalExpired, nil
	case approvaldomain.ApprovedOperationMismatched:
		return nil, domain.ReasonApprovedOperationMismatch, nil
	}
	keyHash := approvaldomain.HashOperationKey(in.ClientID, in.OperationKey)
	if rec.Use != nil && rec.Use.KeyHash != keyHash {
		return nil, domain.ReasonApprovalAlreadyUsed, nil
	}
	return &matchedApproval{requestID: rec.ID, digest: digest, keyHash: keyHash}, "", nil
}

// claimApprovedOperation は照合を通った承認の使用権を得る。並行した別の鍵にはここで負ける。
// 同じ鍵の再送は、結果の不明な要求を資源の冪等性へ委ねるために許可する。
func claimApprovedOperation(ctx context.Context, deps ResourceAccessDeps, approval *matchedApproval, now time.Time) (*ApprovedOperationUse, string, error) {
	claimed, recorded, err := deps.Approvals.ClaimUse(ctx, approval.requestID, approvaldomain.ApprovalUse{KeyHash: approval.keyHash, ClaimedAt: now}, now)
	if err != nil {
		return nil, "", err
	}
	if claimed == nil || claimed.Use == nil {
		return nil, domain.ReasonApprovalExpired, nil
	}
	if claimed.Use.KeyHash != approval.keyHash {
		return nil, domain.ReasonApprovalAlreadyUsed, nil
	}
	return &ApprovedOperationUse{ApprovalRequestID: approval.requestID, OperationDigest: approval.digest, Replayed: !recorded}, "", nil
}

// declaredAttributes は申告を読み、監査のダイジェストとともに返す。読めない申告は nil で、
// 平文を残さずに試行を突き合わせられるよう生の文字列のダイジェストを返す。
func declaredAttributes(raw string) (map[string]any, string) {
	attributes, err := approvaldomain.DecodeStrictJSONObject(raw)
	if err != nil {
		sum := sha256.Sum256([]byte(raw))
		return nil, hex.EncodeToString(sum[:])
	}
	digest, err := approvaldomain.DigestOperationFields(attributes)
	if err != nil {
		return nil, ""
	}
	return attributes, digest
}

// conditionInput は式へ渡す値を作る。値のないクレームはキーを持たず、式が has で区別する。
func conditionInput(token *ports.IntrospectionResult, in ResourceAccessRequest, attributes map[string]any, now time.Time) domain.ConditionInput {
	claims := map[string]any{
		"scope": strings.Fields(token.Scope), "aud": token.Aud, "iat": token.Iat, "exp": token.Exp,
		"delegation_mode": string(domain.DeriveDelegationMode(domain.DelegationSubject{
			Sub: token.Sub, ClientID: token.ClientID, PrincipalType: token.PrincipalType, Act: token.Act,
		})),
	}
	for key, value := range map[string]string{
		"sub": token.Sub, "client_id": token.ClientID, "principal_type": token.PrincipalType, "agent_id": token.AgentID,
	} {
		if value != "" {
			claims[key] = value
		}
	}
	if len(token.AgentChain) > 0 {
		claims["agent_chain"] = token.AgentChain
	}
	return domain.ConditionInput{
		Token: claims, Now: now, Attributes: attributes,
		Operation: map[string]any{"resource_type": in.ResourceType, "resource_id": in.ResourceID, "relation": in.Relation},
	}
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
	ctx context.Context, deps ResourceAccessDeps, tenantID string, token *ports.IntrospectionResult,
) (policyports.PrincipalRef, []policyports.PrincipalRef, bool, error) {
	agents := domain.MergeAgentChain([]string{token.AgentID}, token.AgentChain)
	var subject policyports.PrincipalRef
	if token.PrincipalType == domain.PrincipalTypeAgent && token.AgentID != "" {
		subject = policyports.PrincipalRef{Type: domain.PrincipalTypeAgent, ID: token.AgentID}
	} else {
		user, err := isUserSubject(ctx, deps.Clients, tenantID, token.Sub)
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
		intermediary, err := actIsResourceServer(ctx, deps, tenantID, subject, token.Act)
		if err != nil || !intermediary {
			return policyports.PrincipalRef{}, nil, false, err
		}
	}
	return subject, actors, true, nil
}

// actIsResourceServer は、Agent の主体の act がゲートウェイ一人だけかを判定する。ゲートウェイは
// 執行の中継で権限を足さないので、代行の連鎖として確かめる行為者がいなくても主体を導ける (REQ-OAUTH2-079)。
func actIsResourceServer(ctx context.Context, deps ResourceAccessDeps, tenantID string, subject policyports.PrincipalRef, act map[string]any) (bool, error) {
	clientID, _ := act["sub"].(string)
	if subject.Type != domain.PrincipalTypeAgent || clientID == "" || act["act"] != nil {
		return false, nil
	}
	server, err := deps.Servers.FindByClientID(ctx, tenantID, clientID)
	if err != nil {
		return false, err
	}
	return server != nil && server.IsActive(), nil
}
