// /introspect (RFC 7662) の中核。access_token (JWT) と refresh_token (ストア) の両方を処理。
package usecases

import (
	"context"
	"strings"
	"time"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type IntrospectInput struct {
	Token         string
	TokenTypeHint string // "access_token" | "refresh_token" | ""
}

type IntrospectionResponse struct {
	Active    bool              `json:"active"`
	Scope     string            `json:"scope,omitempty"`
	ClientID  string            `json:"client_id,omitempty"`
	Sub       string            `json:"sub,omitempty"`
	Aud       []string          `json:"aud,omitempty"`
	TokenType string            `json:"token_type,omitempty"`
	Exp       int64             `json:"exp,omitempty"`
	Iat       int64             `json:"iat,omitempty"`
	JTI       string            `json:"jti,omitempty"`
	CNF       map[string]string `json:"cnf,omitempty"`
	Act       map[string]any    `json:"act,omitempty"`
	// AuthorizationDetails は RFC 9396 — RS が信頼する検証点。
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
	// DelegationMode は自律実行と利用者の代理の区別。リソースサーバーが act と
	// principal 種別から各自で導出し直さずに済むよう、導出済みの値を返す。
	DelegationMode domain.DelegationMode `json:"delegation_mode,omitempty"`
}

type IntrospectDeps struct {
	Introspector        ports.TokenIntrospector
	RefreshStore        ports.RefreshTokenStore
	AccessTokenDenylist ports.AccessTokenDenylist
	// AgentRepo は、トークンが発行時に記録した Agent の存在と状態、およびクライアントに
	// 現在束縛された Agent を引く (REQ-OAUTH2-012)。nil なら Agent の判定を行わない。
	AgentRepo agentports.AgentRepository
	// RevocationEpochRepo backs the SCL internal interface CheckRevocationEpoch
	// (docs/modules/sharedsignals/): an access token issued before the
	// Agent's revocation epoch (kill-switch, owner offboard, inbound SET) is
	// fail-closed reported as inactive. nil skips the check.
	RevocationEpochRepo ssports.AgentRevocationEpochRepository
	// Consents は、代行のトークンの発行時の Agent への本人の付与を引く (REQ-OAUTH2-061)。
	// AgentRepo がある配線で nil なら、代行のトークンを失効として扱う。
	Consents ports.ConsentRepository
	// Clients は、サブジェクトが利用者かクライアントかを見分ける。
	Clients ports.OAuth2ClientRepository
}

// AccessTokenIsRevoked は署名検証を通過済みの access token が、その後に失効させられて
// いないかを判定する。判定は 3 つある。
//
//   - jti が AccessTokenDenylist に載っている (RFC 7009 の明示的な失効)。
//   - 発行時の Agent (agent_chain と agent_id) のどれかが削除済みか Active でないか、その
//     revocation epoch が issued_at より後へ前進している (REQ-OAUTH2-012)。
//   - クライアントに現在束縛された Agent の revocation epoch が issued_at より後にある。
//     agent_chain を持たない以前のトークンに、従来の保証を残すための判定である。
//
// 束縛は解除と再束縛で変わるので、発行時の Agent はトークン自身から読む。状態も読むのは、
// 削除した Agent がエポックを残さず、発行の判定と署名の間に止めた Agent では署名の時刻が
// エポックより後になりうるからである。どの読み取りの失敗も error として返し、有効へ倒さない。
// repository が nil の判定は行わない。
//
// IntrospectToken から切り出してあるのは、Bearer を受ける資源サーバー側の経路
// (shared/http/support_http.Authenticator) とトークン交換の入力の検証が、同じ規則を
// 二度実装せずに通すためである (REQ-OAUTH2-047、REQ-OAUTH2-048)。
func AccessTokenIsRevoked(ctx context.Context, deps IntrospectDeps, r *ports.IntrospectionResult) (bool, error) {
	if r == nil || !r.Active {
		return false, nil
	}
	if r.JTI != "" && deps.AccessTokenDenylist != nil {
		revoked, err := deps.AccessTokenDenylist.IsRevoked(ctx, r.JTI)
		if err != nil || revoked {
			return revoked, err
		}
	}
	if deps.AgentRepo == nil {
		return false, nil
	}
	tenantID := tenantports.TenantID(ctx)
	issuedAt := time.Unix(r.Iat, 0)
	for _, agentID := range domain.MergeAgentChain([]string{r.AgentID}, r.AgentChain) {
		agent, err := deps.AgentRepo.FindByID(ctx, tenantID, agentID)
		if err != nil {
			return false, err
		}
		if agent == nil || !agent.IsActive() {
			return true, nil
		}
		if revoked, err := epochSupersedes(ctx, deps, tenantID, agentID, issuedAt); err != nil || revoked {
			return revoked, err
		}
	}
	if withdrawn, err := delegationWithdrawn(ctx, deps, tenantID, r, issuedAt); err != nil || withdrawn {
		return withdrawn, err
	}
	if r.ClientID == "" {
		return false, nil
	}
	bound, err := deps.AgentRepo.FindByClientID(ctx, tenantID, r.ClientID)
	if err != nil || bound == nil {
		return false, err
	}
	return epochSupersedes(ctx, deps, tenantID, bound.ID, issuedAt)
}

// delegationWithdrawn は、利用者を代行するトークンの発行時の Agent のどれかについて、本人の
// 付与がもうトークンを許さないかを判定する (REQ-OAUTH2-061)。失効エポックは Agent 単位で、
// 進めると他の利用者の代行まで止まるので使わない。granted_at より前の発行を無効にするのは、
// 撤回の後の再付与で古いトークンを復活させないためである。
func delegationWithdrawn(ctx context.Context, deps IntrospectDeps, tenantID string, r *ports.IntrospectionResult, issuedAt time.Time) (bool, error) {
	agents := domain.MergeAgentChain([]string{r.AgentID}, r.AgentChain)
	if len(r.Act) == 0 || len(agents) == 0 {
		return false, nil
	}
	user, err := isUserSubject(ctx, deps.Clients, tenantID, r.Sub)
	if err != nil || !user {
		return false, err
	}
	if deps.Consents == nil {
		return true, nil
	}
	now := time.Now().UTC()
	scopes := strings.Fields(r.Scope)
	for _, agentID := range agents {
		grant, err := deps.Consents.FindAgentDelegation(ctx, tenantID, r.Sub, agentID)
		if err != nil {
			return false, err
		}
		if grant == nil || !grant.PermitsDelegation(scopes, now) || issuedAt.Before(grant.GrantedAt.Truncate(time.Second)) {
			return true, nil
		}
	}
	return false, nil
}

// isUserSubject は、トークンのサブジェクトが代行される利用者かを判定する。client_credentials の
// トークンのサブジェクトはクライアント自身で、交換の後は別のクライアントのトークンになるので、
// sub と client_id の比較では見分けられない。登録されたクライアントでなければ利用者として扱い、
// 付与を求める側に倒す。
func isUserSubject(ctx context.Context, clients ports.OAuth2ClientRepository, tenantID, sub string) (bool, error) {
	if sub == "" {
		return false, nil
	}
	if clients == nil {
		return true, nil
	}
	client, err := clients.FindByID(ctx, tenantID, sub)
	if err != nil {
		return false, err
	}
	return client == nil, nil
}

func epochSupersedes(ctx context.Context, deps IntrospectDeps, tenantID, agentID string, issuedAt time.Time) (bool, error) {
	if deps.RevocationEpochRepo == nil {
		return false, nil
	}
	epoch, err := deps.RevocationEpochRepo.FindByAgent(ctx, tenantID, agentID)
	if err != nil {
		return false, err
	}
	return epoch != nil && epoch.Supersedes(issuedAt), nil
}

// RevocationCheckingIntrospector は、署名と期限だけを確かめる TokenIntrospector に
// AccessTokenIsRevoked を重ね、失効したトークンを Active=false として返す。トークン交換の
// subject_token と actor_token は、この形で /introspect と同じ判定を通る (RFC8693-SUBJECT-TOKEN)。
type RevocationCheckingIntrospector struct {
	Deps IntrospectDeps
}

func (i RevocationCheckingIntrospector) IntrospectAccessToken(ctx context.Context, token string) (*ports.IntrospectionResult, error) {
	r, err := i.Deps.Introspector.IntrospectAccessToken(ctx, token)
	if err != nil || r == nil || !r.Active {
		return r, err
	}
	revoked, err := AccessTokenIsRevoked(ctx, i.Deps, r)
	if err != nil {
		return nil, err
	}
	if revoked {
		return &ports.IntrospectionResult{Active: false}, nil
	}
	return r, nil
}

func IntrospectToken(ctx context.Context, deps IntrospectDeps, in IntrospectInput, now time.Time) (*IntrospectionResponse, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	// refresh_token として先に試す（hint=refresh_token か空）
	if in.TokenTypeHint == "" || in.TokenTypeHint == "refresh_token" {
		hash := domain.HashRefreshToken(in.Token)
		rec, err := deps.RefreshStore.FindByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		if rec != nil {
			if rec.TenantID != tenantports.TenantID(ctx) {
				return &IntrospectionResponse{Active: false}, nil
			}
			active := !rec.Revoked && !rec.Rotated && now.Before(rec.AbsoluteExpiresAt)
			if !active {
				return &IntrospectionResponse{Active: false}, nil
			}
			resp := &IntrospectionResponse{
				Active:   true,
				Scope:    strings.Join(rec.Scopes, " "),
				ClientID: rec.ClientID,
				Sub:      rec.UserID,
				Iat:      rec.IssuedAt.Unix(),
				Exp:      rec.ExpiresAt.Unix(),
				JTI:      rec.ID,
				// リフレッシュトークンは act を持たないので direct か autonomous になる。
				// 同じ導出関数を通し、トークン種別ごとに規則が分かれないようにする。
				DelegationMode: domain.DeriveDelegationMode(domain.DelegationSubject{
					Sub: rec.UserID, ClientID: rec.ClientID,
				}),
			}
			if rec.SenderConstraint != nil {
				resp.CNF = map[string]string{}
				if rec.SenderConstraint.JKT != "" {
					resp.CNF["jkt"] = rec.SenderConstraint.JKT
				}
				if rec.SenderConstraint.X5TS256 != "" {
					resp.CNF["x5t#S256"] = rec.SenderConstraint.X5TS256
				}
			}
			return resp, nil
		}
	}
	// access_token として検証
	r, err := deps.Introspector.IntrospectAccessToken(ctx, in.Token)
	if err != nil {
		return nil, err
	}
	revoked, err := AccessTokenIsRevoked(ctx, deps, r)
	if err != nil {
		return nil, err
	}
	if revoked {
		return &IntrospectionResponse{Active: false}, nil
	}
	resp := &IntrospectionResponse{
		Active:               r.Active,
		Scope:                r.Scope,
		ClientID:             r.ClientID,
		Sub:                  r.Sub,
		Aud:                  r.Aud,
		Exp:                  r.Exp,
		Iat:                  r.Iat,
		JTI:                  r.JTI,
		Act:                  r.Act,
		AuthorizationDetails: r.AuthorizationDetails,
	}
	// 立場を語るのは active なトークンだけにする。失効したトークンにモードが付くと、
	// リソースサーバーが active の確認を飛ばして立場だけを読む余地ができる。
	if r.Active {
		// 提示形式は送信者制約から導く。active でないトークンには何も付けない
		// (RFC 7662 §2.2)。制約なしの提示形式は Bearer なので、無条件に代入すると
		// 失効したトークンの応答まで値を運んでしまう。
		resp.TokenType = domain.PresentationTokenType(r.SenderConstraint)
		resp.DelegationMode = domain.DeriveDelegationMode(domain.DelegationSubject{
			Sub: r.Sub, ClientID: r.ClientID, PrincipalType: r.PrincipalType, Act: r.Act,
		})
	}
	if r.SenderConstraint != nil {
		resp.CNF = map[string]string{}
		if r.SenderConstraint.JKT != "" {
			resp.CNF["jkt"] = r.SenderConstraint.JKT
		}
		if r.SenderConstraint.X5TS256 != "" {
			resp.CNF["x5t#S256"] = r.SenderConstraint.X5TS256
		}
	}
	return resp, nil
}
