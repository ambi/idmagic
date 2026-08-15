// Token Exchange Grant (RFC 8693) のユースケース。
//
// 本実装は に忠実に、以下に限定する (fail-closed):
//   - SELF-ISSUED トークンのみ: subject_token / actor_token は本 IdP が発行し、
//     既存の IntrospectAccessToken (署名検証 + active) を通過したものに限る。
//     外部/フェデレーショントークンは対象外 (将来 wi-54 / wi-57)。
//   - DELEGATION ONLY: 発行トークンの sub は subject_token.sub を維持し、必ず act を
//     設定する。impersonation (act 省略 / sub 差し替え) は対象外 (将来、gated)。
//   - MAX DELEGATION DEPTH: act の入れ子はテナントが定める上限まで。超過は invalid_request。
//     上限はテナントが下げられるが、システム既定を超えて上げることはできない。
//     テナントのポリシーを解決できない場合は既定へ退避せず拒否する。
//   - may_act 強制: subject_token に may_act があれば現在アクター sub が may_act.sub と
//     一致しなければ拒否。
//   - RESOURCE INDICATORS (RFC 8707): resource を必須・1 個のみとし、登録済み
//     Active な McpResourceServer に限定する。未登録・Disabled は invalid_target で
//     fail-closed 拒否する。発行トークン aud = [resource]。
//   - REFRESH TOKEN は発行しない。
package usecases

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// DefaultMaxDelegationDepth は発行トークンの act 入れ子の最大深さの既定であり、
// テナント上書きが超えられない上限でもある。値は Tenant 集約が所有し、ここは
// 解決器が組み立てられていない配線 (テストや軽量な組み立て) のための退避先として
// 参照するだけである。定数を二重に持つとどちらかが古くなる。
const DefaultMaxDelegationDepth = tenancydomain.DefaultMaxDelegationDepth

const (
	tokenTypeAccessTokenURN = "urn:ietf:params:oauth:token-type:access_token"
	// tokenTypeJWTURN は外部 workload attestation token (JWT-SVID / Kubernetes
	// projected ServiceAccount token / クラウド instance identity token) を表す
	// subject_token_type (RFC 8693 §3、[[wi-54-workload-identity-federation-spiffe]])。
	tokenTypeJWTURN = "urn:ietf:params:oauth:token-type:jwt"
)

type ExchangeTokenInput struct {
	ClientID             string
	SubjectToken         string
	SubjectTokenType     string
	ActorToken           string
	ActorTokenType       string
	Resource             []string // form の resource (複数指定され得るため slice で受ける)
	Scope                string
	RequestedTokenType   string
	ProofJKT             string
	ProofX5TS256         string
	AuthorizationDetails []spec.AuthorizationDetail // RFC 9396 — 交換で要求する縮小詳細 (任意)
}

type ExchangeTokenResult struct {
	AccessToken          string
	IssuedTokenType      string
	TokenType            string
	ExpiresIn            int
	Scope                string
	AuthorizationDetails []spec.AuthorizationDetail
}

type ExchangeTokenDeps struct {
	ClientRepo            ports.OAuth2ClientRepository
	Introspector          ports.TokenIntrospector
	TokenIssuer           ports.TokenIssuer
	Authorizer            ports.Authorizer
	AuthzDetailTypeRepo   ports.AuthorizationDetailTypeRepository
	McpResourceServerRepo ports.McpResourceServerRepository
	// WorkloadVerifier verifies external workload attestation tokens
	// (subject_token_type=JWT URN) and maps them to a bound Agent's client
	// ([[wi-54-workload-identity-federation-spiffe]]). nil rejects
	// workload subject_token_type as unsupported.
	WorkloadVerifier ports.WorkloadTokenVerifier
	// DelegationPolicy はテナントの act チェーン深さ上限を解決する。nil なら
	// テナント上書きの上限でもある DefaultMaxDelegationDepth を使うため、未配線でも
	// 製品既定より広い許可にはならない。解決器が error を返した場合は交換を拒否する。
	DelegationPolicy ports.DelegationPolicyResolver
	Emit             func(spec.DomainEvent)
}

func ExchangeToken(ctx context.Context, deps ExchangeTokenDeps, in ExchangeTokenInput, now time.Time) (*ExchangeTokenResult, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tenantID := tenancy.TenantID(ctx)

	reject := func(actorUserID string, err *OAuthError) (*ExchangeTokenResult, error) {
		emit(deps.Emit, &domain.TokenExchangeRejected{At: now, TenantID: tenantID, ActorUserID: actorUserID, Reason: err.Code})
		return nil, err
	}

	// --- resource (RFC 8707): 必須・1 個のみ・登録済み Active な McpResourceServer に限定 ---
	if len(nonEmpty(in.Resource)) == 0 {
		return reject("", NewOAuthError("invalid_request", "resource parameter is required"))
	}
	mcpResourceServer, err := ResolveResourceIndicator(ctx, deps.McpResourceServerRepo, tenantID, in.Resource, nil)
	if err != nil {
		if oe, ok := errors.AsType[*OAuthError](err); ok {
			return reject("", oe)
		}
		return nil, err
	}
	resource := mcpResourceServer.Resource
	if in.RequestedTokenType != "" && in.RequestedTokenType != tokenTypeAccessTokenURN {
		return reject("", NewOAuthError("invalid_request", "unsupported requested_token_type"))
	}

	// --- subject_token (必須) ---
	if in.SubjectToken == "" {
		return reject("", NewOAuthError("invalid_request", "subject_token is required"))
	}
	var subject *ports.IntrospectionResult
	var workloadGrant *workloaddomain.WorkloadIdentityGrant
	switch in.SubjectTokenType {
	case "", tokenTypeAccessTokenURN:
		// SELF-ISSUED access_token: 本 IdP が発行し IntrospectAccessToken を通過した
		// トークンの委任。
		subject, err = deps.Introspector.IntrospectAccessToken(ctx, in.SubjectToken)
		if err != nil {
			return nil, err
		}
		if subject == nil || !subject.Active {
			return reject("", NewOAuthError("invalid_grant", "The subject_token is invalid or expired."))
		}
	case tokenTypeJWTURN:
		// 外部 workload attestation (JWT-SVID 等) — WorkloadIdentity の
		// VerifyWorkloadAttestation を経由し、束縛先 Agent の client へ写す。
		// 専用の資格情報経路は新設せず、以降のロジックは introspection
		// 結果と同じ形へ正規化して再利用する。
		if deps.WorkloadVerifier == nil {
			return reject("", NewOAuthError("invalid_request", "unsupported subject_token_type"))
		}
		grant, werr := deps.WorkloadVerifier.VerifyWorkloadToken(ctx, tenantID, in.SubjectToken, now)
		if werr != nil {
			return reject("", NewOAuthError("invalid_grant", "The subject_token is invalid or expired."))
		}
		workloadClient, cerr := deps.ClientRepo.FindByID(ctx, tenantID, grant.ClientID)
		if cerr != nil {
			return nil, cerr
		}
		if workloadClient == nil {
			return reject("", NewOAuthError("invalid_grant", "workload identity is not bound to a usable client"))
		}
		workloadGrant = grant
		subject = &ports.IntrospectionResult{Active: true, Sub: workloadClient.ClientID, Scope: workloadClient.Scope}
	default:
		return reject("", NewOAuthError("invalid_request", "unsupported subject_token_type"))
	}

	// --- actor_token (任意) ---
	currentActorSub := in.ClientID
	if in.ActorToken != "" {
		if in.ActorTokenType != "" && in.ActorTokenType != tokenTypeAccessTokenURN {
			return reject("", NewOAuthError("invalid_request", "unsupported actor_token_type"))
		}
		actor, err := deps.Introspector.IntrospectAccessToken(ctx, in.ActorToken)
		if err != nil {
			return nil, err
		}
		if actor == nil || !actor.Active {
			return reject("", NewOAuthError("invalid_grant", "actor_token is invalid or revoked"))
		}
		currentActorSub = actor.Sub
	}
	if currentActorSub == "" {
		return reject("", NewOAuthError("invalid_request", "The current actor cannot be determined."))
	}

	// --- may_act 強制 (fail-closed) ---
	if subject.MayAct != nil {
		mayActSub, _ := subject.MayAct["sub"].(string)
		if mayActSub == "" || mayActSub != currentActorSub {
			return reject(currentActorSub, NewOAuthError("invalid_grant", "current actor is not allowed by may_act"))
		}
	}

	// --- act チェーン構築 (RFC 8693 §4.1) ---
	// act = {"sub": currentActor}; subject_token に act があれば入れ子で連結する。
	act := map[string]any{"sub": currentActorSub}
	if subject.Act != nil {
		act["act"] = subject.Act
	}
	depth := actDepth(act)
	maxDepth, err := resolveMaxDelegationDepth(ctx, deps.DelegationPolicy, tenantID)
	if err != nil {
		// テナントのポリシーを読めないまま既定へ退避すると、テナントが下げたはずの
		// 上限が黙って戻る。拒否は監査へ残るので、設定不備は運用側から見える。
		return reject(currentActorSub, NewOAuthError("invalid_request", "The delegation policy could not be resolved."))
	}
	if depth > maxDepth {
		return reject(currentActorSub, NewOAuthError("invalid_request", "The delegation depth exceeds the limit."))
	}

	// --- scope ダウンスコープ (拡大不可) ---
	subjectScopes := strings.Fields(subject.Scope)
	grantedScopes := subjectScopes
	if requested := strings.Fields(in.Scope); len(requested) > 0 {
		subset := map[string]bool{}
		for _, s := range subjectScopes {
			subset[s] = true
		}
		for _, s := range requested {
			if !subset[s] {
				return reject(currentActorSub, NewOAuthError("invalid_scope", "The requested scope exceeds the subject_token scope."))
			}
		}
		grantedScopes = requested
	}
	for _, s := range grantedScopes {
		if !slices.Contains(mcpResourceServer.Scopes, s) {
			return reject(currentActorSub, NewOAuthError("invalid_scope", "request exceeds allowed scope of the resource"))
		}
	}
	if slices.ContainsFunc(grantedScopes, func(scope string) bool { return strings.HasPrefix(scope, "account:") }) &&
		(subject.Sub == "" || subject.Sub == subject.ClientID) {
		return reject(currentActorSub, NewOAuthError("invalid_scope", "account scope requires user subject"))
	}

	// --- authorization_details ダウンスコープ (拡大不可, RFC 9396) ---
	// 要求があれば登録 type に対し検証し、subject_token の詳細の部分集合に限る。
	// 要求が無ければ subject の詳細を保持する (縮小のみ、決して拡張しない)。
	grantedDetails := subject.AuthorizationDetails
	if len(in.AuthorizationDetails) > 0 {
		if err := ValidateAuthorizationDetails(ctx, deps.AuthzDetailTypeRepo, in.AuthorizationDetails); err != nil {
			return reject(currentActorSub, NewOAuthError("invalid_authorization_details", err.Error()))
		}
		types, err := LoadAuthorizationDetailTypes(ctx, deps.AuthzDetailTypeRepo, in.AuthorizationDetails)
		if err != nil {
			return nil, err
		}
		if err := domain.DetailsSubsetOf(in.AuthorizationDetails, subject.AuthorizationDetails, types); err != nil {
			return reject(currentActorSub, NewOAuthError("invalid_authorization_details", "request exceeds authorization_details of subject_token"))
		}
		grantedDetails = in.AuthorizationDetails
	}

	// --- client の解決 (AuthZEN gate は token_handler 側の grant 宣言チェックで担保) ---
	client, err := deps.ClientRepo.FindByID(ctx, tenantID, in.ClientID)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return reject(currentActorSub, NewOAuthError("invalid_client", "unknown client_id"))
	}
	d, err := evaluateTokenExchangePolicy(ctx, deps.Authorizer, client, currentActorSub, subject.Sub, resource, grantedScopes, depth, now)
	if err != nil {
		return nil, err
	}
	if !d.Permit {
		return reject(currentActorSub, NewOAuthError("invalid_grant", "Token exchange was rejected: "+strings.Join(d.Reasons, ", ")))
	}

	// --- sender constraint (DPoP / mTLS) ---
	var sc *domain.SenderConstraint
	if in.ProofJKT != "" {
		sc = &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: in.ProofJKT}
	} else if in.ProofX5TS256 != "" {
		sc = &domain.SenderConstraint{Type: spec.SenderConstraintMTLS, X5TS256: in.ProofX5TS256}
	}

	// --- 発行 (DELEGATION ONLY: sub = subject.sub, aud = [resource], act 必須) ---
	var agentID string
	if workloadGrant != nil {
		agentID = workloadGrant.AgentID
	}
	access, jti, err := deps.TokenIssuer.SignAccessToken(ctx, ports.AccessTokenInput{
		Client: client, Sub: subject.Sub, Scopes: grantedScopes,
		SenderConstraint: sc, AuthTime: now.Unix(),
		Audiences: []string{resource}, Act: act,
		AuthorizationDetails: grantedDetails, AgentID: agentID,
	})
	if err != nil {
		return nil, err
	}

	emit(deps.Emit, &domain.AccessTokenIssued{
		At: now, TenantID: tenantID, JTI: jti, ClientID: client.ClientID,
		UserID: subject.Sub, Scopes: grantedScopes, SenderConstraint: senderConstraintTag(sc),
	})
	// 委譲モードは introspection と同じ導出関数を通す。ここで別の規則を書くと、
	// 監査とリソースサーバーの見え方が食い違い、しかもその不整合は調査のときに
	// 最も見つけにくい形で現れる。
	principalType := ""
	if agentID != "" {
		principalType = domain.PrincipalTypeAgent
	}
	emit(deps.Emit, &domain.TokenExchanged{
		At: now, TenantID: tenantID, ActorUserID: currentActorSub, SubjectUserID: subject.Sub,
		Audience: resource, DelegationDepth: depth, MaxDelegationDepth: maxDepth,
		DelegationMode: domain.DeriveDelegationMode(domain.DelegationSubject{
			Sub: subject.Sub, ClientID: client.ClientID, PrincipalType: principalType, Act: act,
		}),
	})
	if workloadGrant != nil {
		emit(deps.Emit, &workloaddomain.WorkloadTokenExchanged{
			At: now, TenantID: tenantID, TrustBundleID: workloadGrant.TrustBundleID,
			BindingID: workloadGrant.BindingID, AgentID: workloadGrant.AgentID, Audience: resource,
		})
	}

	tokenType := "Bearer"
	if sc != nil && sc.Type == spec.SenderConstraintDPoP {
		tokenType = "DPoP"
	}
	return &ExchangeTokenResult{
		AccessToken:          access,
		IssuedTokenType:      tokenTypeAccessTokenURN,
		TokenType:            tokenType,
		ExpiresIn:            deps.TokenIssuer.AccessTokenTTLSeconds(),
		Scope:                strings.Join(grantedScopes, " "),
		AuthorizationDetails: grantedDetails,
	}, nil
}

func evaluateTokenExchangePolicy(
	ctx context.Context,
	authorizer ports.Authorizer,
	client *domain.OAuth2Client,
	actorUserID, subjectUserID, resource string,
	scopes []string,
	delegationDepth int,
	now time.Time,
) (spec.AuthZResponse, error) {
	req := spec.AuthZRequest{
		Subject: spec.AuthZSubject{
			Type: "Client",
			ID:   client.ClientID,
			Properties: spec.AuthZSubjectProps{
				ClientType: client.ClientType,
				GrantTypes: slices.Clone(client.GrantTypes),
				Scopes:     strings.Fields(client.Scope),
				TenantID:   client.TenantID,
			},
		},
		Action: spec.ActionTokenGrantTokenExchange,
		Resource: spec.AuthZResource{
			Type: "TokenExchange",
			ID:   resource,
			Properties: spec.AuthZResourceProps{
				Scopes:   slices.Clone(scopes),
				TenantID: client.TenantID,
			},
		},
		Context: spec.AuthZContext{
			ActorUserID: actorUserID, SubjectUserID: subjectUserID, Audience: resource,
			DelegationDepth: delegationDepth, Now: now,
		},
	}
	if authorizer == nil {
		return spec.Evaluate(req), nil
	}
	return authorizer.Authorize(ctx, req)
}

// resolveMaxDelegationDepth はテナントの委譲深さ上限を解決する。解決器が組み立てられて
// いない配線では既定 (= 上書きが超えられない上限) を使い、解決に失敗した場合は error を
// 返して呼び出し側に拒否させる。解決できた値が既定を超えていた場合も既定へ丸める:
// 上書きは厳しい方向にのみ働くという規則を、保存側と評価側の両方で守る。
func resolveMaxDelegationDepth(ctx context.Context, resolver ports.DelegationPolicyResolver, tenantID string) (int, error) {
	if resolver == nil {
		return DefaultMaxDelegationDepth, nil
	}
	resolved, err := resolver.MaxDelegationDepth(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	if resolved < 1 || resolved > DefaultMaxDelegationDepth {
		return DefaultMaxDelegationDepth, nil
	}
	return resolved, nil
}

// actDepth は act claim の入れ子の深さを数える。最も外側の act を 1 とする。
func actDepth(act map[string]any) int {
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

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}
