// Package token_oauth2 は OAuth2 のイントロスペクションの結果を TokenVault の呼び出し元の
// トークンへ変換する。検証と失効の規則は OAuth2 が所有する。
package token_oauth2

import (
	"context"
	"slices"
	"time"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// Verifier は ports.TokenVerifier の実装である。Introspector は失効の判定を含む。
type Verifier struct {
	Introspector oauthports.TokenIntrospector
	Replay       oauthports.DpopReplayStore
}

// Verify はトークンと DPoP の証明を確かめる。証明の jti はトークンが有効なときだけ記録する。
func (v Verifier) Verify(ctx context.Context, credential ports.CallerCredential) (*domain.CallerToken, error) {
	result, err := v.Introspector.IntrospectAccessToken(ctx, credential.Token)
	if err != nil {
		return nil, err
	}
	if result == nil || !result.Active || result.Sub == "" {
		return nil, nil //nolint:nilnil // 無効なトークンは nil で表す。
	}
	// 別の資源へ宛てたトークンを受けると、その資源のサーバーが Vault を呼べてしまう。
	if !slices.Contains(result.Aud, credential.Resource) {
		return nil, nil //nolint:nilnil // 宛先の異なるトークンは無効なトークンとして扱う。
	}
	if constraint := result.SenderConstraint; constraint != nil {
		if constraint.Type != spec.SenderConstraintDPoP || credential.Scheme != "dpop" || v.Replay == nil {
			return nil, nil //nolint:nilnil // 証明できない送信者制約は無効なトークンとして扱う。
		}
		proof, err := tokens_jose.VerifyDPoPForResource(
			ctx, credential.DPoPProof, credential.Method, credential.HTU, credential.Token, v.Replay, time.Now().UTC(),
		)
		if err != nil || proof == nil || proof.JKT != constraint.JKT {
			return nil, nil //nolint:nilnil,nilerr // 証明の不一致は無効なトークンとして扱う。
		}
	}
	return &domain.CallerToken{
		Delegated: len(result.Act) > 0, Subject: result.Sub, PrincipalType: result.PrincipalType, AgentID: result.AgentID,
		AgentChain: result.AgentChain, AuthorizationDetails: result.AuthorizationDetails,
	}, nil
}

var _ ports.TokenVerifier = Verifier{}
