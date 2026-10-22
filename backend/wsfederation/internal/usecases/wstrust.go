package usecases

import (
	"context"
	"net/http"
	"strings"
	"time"

	claimports "github.com/ambi/idmagic/backend/claimmapping/ports"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

// WsTrustService は WS-Trust active STS のトークン発行判断 (アプリケーションの関門・claim 発行・
// token type 解決) を所有する。SOAP body 読取・replay 検査・throttle・資格情報検証は adapter が担い、
// 本サービスは認証済み利用者と解決済み RP を前提に発行内容を決める。
type WsTrustService struct {
	Gate ApplicationGate
	Emit func(spec.DomainEvent)
}

// TokenRequest は認証・RP 解決を通過した後のトークン発行入力。
type TokenRequest struct {
	RP                 feddomain.WsFedRelyingParty
	User               userdomain.User
	RequestedTokenType string // RST の要求 token type (空なら RP 既定)。
	// AuthenticatedAt は UsernameToken のパスワードを照合した時刻。関門へパスワードだけで
	// 認証した事実として渡す。
	AuthenticatedAt time.Time
	ClientIP        string
}

// passwordAuthentication は UsernameToken のパスワードだけで認証した事実である。WS-Trust には
// 追加の認証の手段がないので、MFA を求めるサインインポリシーはこれを満たさない。
func passwordAuthentication(sub string, at time.Time) *authdomain.AuthenticationContext {
	amr := []string{"pwd"}
	return &authdomain.AuthenticationContext{UserID: sub, AMR: amr, ACR: authdomain.DeriveACR(amr), AuthTime: at.Unix()}
}

// TokenDecision はトークン発行判断の結果。RejectReason が非空なら発行拒否。
type TokenDecision struct {
	ClaimResult claimdomain.ClaimIssuanceResult
	// Issuer は Assertion の発行者。空ならテナントの発行者を使う。
	Issuer       string
	TokenType    feddomain.WsFedTokenType
	RejectReason string // 非空なら発行拒否 (WsTrustTokenRejected を発行し RejectStatus を返す)。
	RejectStatus int
}

// IssueToken は claim を発行し、要求 token type を検証して有効 token type を確定する。
// 挙動は旧 HTTP ハンドラ handleWsTrustUsernameMixed の claim / token type 決定部と一致する。
// attrSchemaRepo は attribute visibility floor を強制するための tenant custom
// attribute schema 解決に使う。nil なら builtin 定義のみを floor とする。
func (s WsTrustService) IssueToken(ctx context.Context, tenantID string, attrSchemaRepo claimports.TenantAttributeSchemaRepo, req TokenRequest) (TokenDecision, error) {
	reason, err := enforceApplicationAccess(ctx, s.Gate, s.Emit, tenantID, req.RP.Wtrealm, req.User.ID,
		passwordAuthentication(req.User.ID, req.AuthenticatedAt), req.ClientIP, req.AuthenticatedAt)
	if err != nil {
		return TokenDecision{}, err
	}
	if reason != "" {
		return TokenDecision{RejectReason: reason, RejectStatus: http.StatusForbidden}, nil
	}
	issued, reject, err := issueForRelyingParty(ctx, tenantID, attrSchemaRepo, req.RP, req.User)
	if err != nil {
		return TokenDecision{}, err
	}
	if reject != "" {
		return TokenDecision{RejectReason: reject, RejectStatus: http.StatusInternalServerError}, nil
	}
	tokenType := req.RP.EffectiveTokenType()
	if strings.TrimSpace(req.RequestedTokenType) != "" {
		if req.RequestedTokenType != string(feddomain.TokenTypeSAML11) && req.RequestedTokenType != string(feddomain.TokenTypeSAML20) {
			return TokenDecision{RejectReason: "unsupported token type", RejectStatus: 400}, nil
		}
		tokenType = feddomain.WsFedTokenType(req.RequestedTokenType)
	}
	return TokenDecision{ClaimResult: issued.claims, Issuer: issued.issuer, TokenType: tokenType}, nil
}
