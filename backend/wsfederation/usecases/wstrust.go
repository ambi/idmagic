package usecases

import (
	"strings"

	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

// WsTrustService は WS-Trust active STS のトークン発行判断 (claim 発行・token type 解決) を所有する。
// SOAP body 読取・replay 検査・throttle・資格情報検証は adapter が担い、本サービスは
// 認証済み利用者と解決済み RP を前提に発行内容を決める。
type WsTrustService struct{}

// TokenRequest は認証・RP 解決を通過した後のトークン発行入力。
type TokenRequest struct {
	RP                 feddomain.WsFedRelyingParty
	User               spec.User
	RequestedTokenType string // RST の要求 token type (空なら RP 既定)。
}

// TokenDecision はトークン発行判断の結果。RejectReason が非空なら発行拒否。
type TokenDecision struct {
	ClaimResult  feddomain.ClaimIssuanceResult
	TokenType    feddomain.WsFedTokenType
	RejectReason string // 非空なら発行拒否 (WsTrustTokenRejected を発行し RejectStatus を返す)。
	RejectStatus int
}

// IssueToken は claim を発行し、要求 token type を検証して有効 token type を確定する。
// 挙動は旧 HTTP ハンドラ handleWsTrustUsernameMixed の claim / token type 決定部と一致する。
func (WsTrustService) IssueToken(req TokenRequest) TokenDecision {
	attrs, err := feddomain.ApplyEntraProfile(feddomain.ResolveUserAttributes(req.User), req.RP.EntraProfile)
	if err != nil {
		return TokenDecision{RejectReason: "entra profile failed", RejectStatus: 500}
	}
	result, err := feddomain.IssueClaims(req.RP.ClaimPolicy, attrs)
	if err != nil {
		return TokenDecision{RejectReason: "claim issuance failed", RejectStatus: 500}
	}
	tokenType := req.RP.EffectiveTokenType()
	if strings.TrimSpace(req.RequestedTokenType) != "" {
		if req.RequestedTokenType != string(feddomain.TokenTypeSAML11) && req.RequestedTokenType != string(feddomain.TokenTypeSAML20) {
			return TokenDecision{RejectReason: "unsupported token type", RejectStatus: 400}
		}
		tokenType = feddomain.WsFedTokenType(req.RequestedTokenType)
	}
	return TokenDecision{ClaimResult: result, TokenType: tokenType}
}
