package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// IntrospectionResult は RFC 7662 のレスポンス。
type IntrospectionResult struct {
	Active           bool
	JTI              string
	ClientID         string
	Sub              string
	Scope            string
	Exp              int64
	Iat              int64
	Managed          bool
	SenderConstraint *domain.SenderConstraint
	// Aud / Act / MayAct は RFC 8693 トークン交換のために検証済みペイロードから抽出する。
	// Aud は単一文字列でも配列でも常に []string に正規化する。
	Aud    []string
	Act    map[string]any
	MayAct map[string]any
	// AuthorizationDetails は RFC 9396 の構造化詳細。検証済みペイロードから
	// 抽出し、introspection 応答とトークン交換のダウンスコープ判定に使う。
	AuthorizationDetails []spec.AuthorizationDetail
	// PrincipalType / AgentID は Agent に束縛されたトークンの標識 (wi-49)。
	// 委譲モードの導出は act チェーンだけでは足りず、subject が非人間の
	// プリンシパルかどうかをここで見る。
	PrincipalType string
	AgentID       string
	// AgentChain は発行時に権限を支えた Agent (agent_chain claim, REQ-OAUTH2-012)。
	// 失効判定は現在の束縛ではなく、この集合と AgentID の Agent を読む。
	AgentChain []string
	// ApprovalRequestID は CIBA の承認から発行したトークンの承認リクエスト (REQ-OAUTH2-065)。
	ApprovalRequestID string
}

type TokenIntrospector interface {
	IntrospectAccessToken(ctx context.Context, token string) (*IntrospectionResult, error)
}
