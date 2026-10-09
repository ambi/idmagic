package handlers_http

import (
	"context"
	"errors"
	"time"

	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
)

var (
	errDPoPReplayStoreUnavailable = errors.New("dpop replay store is not configured")
	errDPoPProofMissing           = errors.New("dpop proof is missing")
)

// ResourceAccessTokens は、共有の HTTP 支援の Bearer 認証へ、イントロスペクション、失効の
// 判定、DPoP の検証を渡す。管理 API と account API は、/introspect と同じ規則で
// アクセストークンを受ける。
type ResourceAccessTokens struct {
	Introspector oauthports.TokenIntrospector
	// Revocation は /introspect と同じ失効判定 (AccessTokenDenylist と AgentRevocationEpoch)
	// の repository 群 (REQ-OAUTH2-047)。判定そのものは tokenusecases.AccessTokenIsRevoked が
	// 一手に持つ。ゼロ値では判定を行わないので、SharedSignals や denylist を組み立てない
	// 軽量な配線でもそのまま動く。
	Revocation      tokenusecases.IntrospectDeps
	DpopReplayStore oauthports.DpopReplayStore
}

// IntrospectAccessToken は、有効で失効していないアクセストークンを返す。
func (r ResourceAccessTokens) IntrospectAccessToken(ctx context.Context, token string) (*support.AccessToken, error) {
	res, err := r.Introspector.IntrospectAccessToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if res == nil || !res.Active || res.Sub == "" {
		return nil, nil //nolint:nilnil // 無効なトークンは nil で表し、拒否の種類は呼び出し元が決める。
	}
	revoked, err := tokenusecases.AccessTokenIsRevoked(ctx, r.Revocation, res)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, nil //nolint:nilnil // 失効したトークンは無効なトークンと同じく nil で表す。
	}
	verified := &support.AccessToken{
		Subject: res.Sub, ClientID: res.ClientID, Scope: res.Scope, Audience: res.Aud,
		IssuedAt: res.Iat, Managed: res.Managed,
	}
	if res.SenderConstraint != nil && res.SenderConstraint.Type == spec.SenderConstraintDPoP {
		verified.DPoPBound = true
		verified.DPoPJKT = res.SenderConstraint.JKT
	}
	return verified, nil
}

// VerifyDPoPProof は、保護されたリソースへの DPoP の証明を検証し、鍵の拇印を返す。
// 証明の jti はリプレイ記録へ書き込む。
func (r ResourceAccessTokens) VerifyDPoPProof(ctx context.Context, proof support.DPoPProof) (string, error) {
	if r.DpopReplayStore == nil {
		return "", errDPoPReplayStoreUnavailable
	}
	verified, err := tokens_jose.VerifyDPoPForResource(
		ctx, proof.Header, proof.Method, proof.HTU, proof.AccessToken, r.DpopReplayStore, time.Now().UTC(),
	)
	if err != nil {
		return "", err
	}
	if verified == nil {
		return "", errDPoPProofMissing
	}
	return verified.JKT, nil
}
