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
	"github.com/ambi/idmagic/backend/tenancy"
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
	// AgentRepo resolves the Agent bound to an access token's client_id
	// , so RevocationEpochRepo can fail-closed check its issued_at
	// against the Agent's revocation epoch (wi-58). nil skips the
	// check.
	AgentRepo agentports.AgentRepository
	// RevocationEpochRepo backs the SCL internal interface CheckRevocationEpoch
	// (spec/contexts/sharedsignals.yaml): an access token issued before the
	// Agent's revocation epoch (kill-switch, owner offboard, inbound SET) is
	// fail-closed reported as inactive. nil skips the check.
	RevocationEpochRepo ssports.AgentRevocationEpochRepository
}

// AccessTokenIsRevoked は署名検証を通過済みの access token が、その後に失効させられて
// いないかを判定する。判定は 2 つある。jti が AccessTokenDenylist に載っていること
// (RFC 7009 の明示的な失効) と、token の subject である Agent の revocation epoch が
// issued_at より後へ前進していること (kill-switch / 所有者オフボード / 受理済み inbound SET)
// である。どちらも repository が nil なら判定を行わない。
//
// IntrospectToken から切り出してあるのは、Bearer を受ける資源サーバー側の経路
// (shared/http/support_http.Authenticator) が、同じ規則を二度実装せずに通すためである
// (REQ-OAUTH2-047)。失効判定を通らない access token 検証経路を残さない。
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
	if r.ClientID == "" || deps.AgentRepo == nil || deps.RevocationEpochRepo == nil {
		return false, nil
	}
	tenantID := tenancy.TenantID(ctx)
	agent, err := deps.AgentRepo.FindByClientID(ctx, tenantID, r.ClientID)
	if err != nil || agent == nil {
		return false, err
	}
	epoch, err := deps.RevocationEpochRepo.FindByAgent(ctx, tenantID, agent.ID)
	if err != nil {
		return false, err
	}
	return epoch != nil && epoch.Supersedes(time.Unix(r.Iat, 0)), nil
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
			if rec.TenantID != tenancy.TenantID(ctx) {
				return &IntrospectionResponse{Active: false}, nil
			}
			active := !rec.Revoked && !rec.Rotated && now.Before(rec.AbsoluteExpiresAt)
			if !active {
				return &IntrospectionResponse{Active: false}, nil
			}
			resp := &IntrospectionResponse{
				Active:    true,
				Scope:     strings.Join(rec.Scopes, " "),
				ClientID:  rec.ClientID,
				Sub:       rec.UserID,
				TokenType: "refresh_token",
				Iat:       rec.IssuedAt.Unix(),
				Exp:       rec.ExpiresAt.Unix(),
				JTI:       rec.ID,
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
		TokenType:            r.TokenType,
		Exp:                  r.Exp,
		Iat:                  r.Iat,
		JTI:                  r.JTI,
		Act:                  r.Act,
		AuthorizationDetails: r.AuthorizationDetails,
	}
	// 立場を語るのは active なトークンだけにする。失効したトークンにモードが付くと、
	// リソースサーバーが active の確認を飛ばして立場だけを読む余地ができる。
	if r.Active {
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
