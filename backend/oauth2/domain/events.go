package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type AuthorizationDetailsRequested struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ClientID    string    `json:"clientId"`
	UserID      string    `json:"userId,omitempty"`
	DetailTypes []string  `json:"detailTypes"`
}

func (e *AuthorizationDetailsRequested) EventType() string     { return "AuthorizationDetailsRequested" }
func (e *AuthorizationDetailsRequested) OccurredAt() time.Time { return e.At }

type AuthorizationDetailsConsented struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	UserID      string    `json:"userId"`
	ClientID    string    `json:"clientId"`
	DetailTypes []string  `json:"detailTypes"`
}

func (e *AuthorizationDetailsConsented) EventType() string     { return "AuthorizationDetailsConsented" }
func (e *AuthorizationDetailsConsented) OccurredAt() time.Time { return e.At }

type AuthorizationDetailsRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Reason   string    `json:"reason"`
}

func (e *AuthorizationDetailsRejected) EventType() string     { return "AuthorizationDetailsRejected" }
func (e *AuthorizationDetailsRejected) OccurredAt() time.Time { return e.At }

type AuthorizationCodeIssued struct {
	At                  time.Time                `json:"-"`
	TenantID            string                   `json:"tenantId"`
	ClientID            string                   `json:"clientId"`
	UserID              string                   `json:"userId"`
	Scopes              []string                 `json:"scopes"`
	CodeChallengeMethod spec.CodeChallengeMethod `json:"codeChallengeMethod"`
}

func (e *AuthorizationCodeIssued) EventType() string     { return "AuthorizationCodeIssued" }
func (e *AuthorizationCodeIssued) OccurredAt() time.Time { return e.At }

type AuthorizationCodeRedeemed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *AuthorizationCodeRedeemed) EventType() string     { return "AuthorizationCodeRedeemed" }
func (e *AuthorizationCodeRedeemed) OccurredAt() time.Time { return e.At }

type AccessTokenIssued struct {
	At               time.Time `json:"-"`
	TenantID         string    `json:"tenantId"`
	JTI              string    `json:"jti"`
	ClientID         string    `json:"clientId"`
	UserID           string    `json:"userId"`
	Scopes           []string  `json:"scopes"`
	SenderConstraint string    `json:"senderConstraint"` // "none" | "dpop" | "mtls"
	// AgentID はトークンを要求したクライアントに結び付いた Agent。Agent の棚卸しが最終の
	// 発行時刻を監査の agent.id 軸から読む (REQ-IDMANAGEMENT-094)。
	AgentID string `json:"agentId,omitempty"`
}

func (e *AccessTokenIssued) EventType() string     { return "AccessTokenIssued" }
func (e *AccessTokenIssued) OccurredAt() time.Time { return e.At }

type RefreshTokenIssued struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	TokenID  string    `json:"tokenId"`
	FamilyID string    `json:"familyId"`
	ParentID string    `json:"parentId,omitempty"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *RefreshTokenIssued) EventType() string     { return "RefreshTokenIssued" }
func (e *RefreshTokenIssued) OccurredAt() time.Time { return e.At }

type RefreshTokenRotated struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	OldTokenID string    `json:"oldTokenId"`
	NewTokenID string    `json:"newTokenId"`
	FamilyID   string    `json:"familyId"`
}

func (e *RefreshTokenRotated) EventType() string     { return "RefreshTokenRotated" }
func (e *RefreshTokenRotated) OccurredAt() time.Time { return e.At }

type TokenRevoked struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	TokenType string    `json:"tokenType"` // "access_token" | "refresh_token"
	TokenID   string    `json:"tokenId"`
	Reason    string    `json:"reason"`
}

func (e *TokenRevoked) EventType() string     { return "TokenRevoked" }
func (e *TokenRevoked) OccurredAt() time.Time { return e.At }

type TokenIntrospected struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	RSClientID string    `json:"rsClientId"`
	TokenID    string    `json:"tokenId"`
	Active     bool      `json:"active"`
}

func (e *TokenIntrospected) EventType() string     { return "TokenIntrospected" }
func (e *TokenIntrospected) OccurredAt() time.Time { return e.At }

type TokenExchanged struct {
	At            time.Time `json:"-"`
	TenantID      string    `json:"tenantId"`
	ActorUserID   string    `json:"actorUserId"`
	SubjectUserID string    `json:"subjectUserId"`
	Audience      string    `json:"audience"`
	// DelegationDepth は発行トークンの act 入れ子の深さ。
	DelegationDepth int `json:"delegationDepth"`
	// MaxDelegationDepth は判定に適用した上限。深さだけを残すと、後から
	// 「その時どの上限で通したのか」を再現できない。
	MaxDelegationDepth int `json:"maxDelegationDepth"`
	// DelegationMode は発行トークンの委譲モード。introspection と同じ導出関数を通す。
	DelegationMode DelegationMode `json:"delegationMode"`
	// AgentID は交換したクライアントに束縛された Agent。監査の actor.type / agent.id が
	// 行為者の種別を推測ではなくイベントから読めるようにする。
	AgentID string `json:"agentId,omitempty"`
	// ActorChain は発行トークンの act チェーンに現れる主体を外側から順に並べたもの。
	// 深さと両端だけでは中間の行為者を辿れないため、参加者そのものを残す。
	ActorChain []string `json:"actorChain"`
}

func (e *TokenExchanged) EventType() string     { return "TokenExchanged" }
func (e *TokenExchanged) OccurredAt() time.Time { return e.At }

type TokenExchangeRejected struct {
	At          time.Time `json:"-"`
	TenantID    string    `json:"tenantId"`
	ActorUserID string    `json:"actorUserId,omitempty"`
	Reason      string    `json:"reason"`
}

func (e *TokenExchangeRejected) EventType() string     { return "TokenExchangeRejected" }
func (e *TokenExchangeRejected) OccurredAt() time.Time { return e.At }

// AgentApprovalRequired は Agent の区分が人間の承認を要求するため、承認を記録しない
// 発行経路を fail-closed で拒否した (REQ-OAUTH2-050)。client_credentials、トークン
// 交換、ワークロード ID 連携による交換が対象で、承認を伴う CIBA 経路の結果は
// BackchannelAuthApproved が表す。
//
// Kind は解決した値をそのまま残す。既知の列挙に無い値こそフェイルクローズ規則が
// 存在する理由であり、丸めた値を記録すると判断の再現ができなくなる。
type AgentApprovalRequired struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	AgentID   string    `json:"agentId"`
	ClientID  string    `json:"clientId"`
	Kind      string    `json:"kind"`
	GrantType string    `json:"grantType"`
}

func (e *AgentApprovalRequired) EventType() string     { return "AgentApprovalRequired" }
func (e *AgentApprovalRequired) OccurredAt() time.Time { return e.At }

// AgentClientAuthenticationRejected は Agent に束縛したクライアントの認証方式が鍵の
// 所有を証明しないため、トークン要求をグラントによらず拒否した (REQ-OAUTH2-055)。
// 応答は束縛の有無を区別しない invalid_client なので、拒否の理由はここにだけ残る。
type AgentClientAuthenticationRejected struct {
	At                      time.Time `json:"-"`
	TenantID                string    `json:"tenantId"`
	AgentID                 string    `json:"agentId"`
	ClientID                string    `json:"clientId"`
	TokenEndpointAuthMethod string    `json:"tokenEndpointAuthMethod"`
	GrantType               string    `json:"grantType"`
}

func (e *AgentClientAuthenticationRejected) EventType() string {
	return "AgentClientAuthenticationRejected"
}
func (e *AgentClientAuthenticationRejected) OccurredAt() time.Time { return e.At }

// ProtectedResourceMetadataServed は RFC 9728 — /.well-known/oauth-protected-resource
// が登録済み McpResourceServer の metadata を配信した。
type ProtectedResourceMetadataServed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Resource string    `json:"resource"`
}

func (e *ProtectedResourceMetadataServed) EventType() string {
	return "ProtectedResourceMetadataServed"
}
func (e *ProtectedResourceMetadataServed) OccurredAt() time.Time { return e.At }

// ResourceScopedTokenIssued は RFC 8707 — resource indicator に基づき audience を単一
// McpResourceServer へ厳格限定した Access Token を発行した。Authorize/PAR/
// Token(authorization_code) 経路が対象。token-exchange は TokenExchanged.audience で表現する。
type ResourceScopedTokenIssued struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Resource string    `json:"resource"`
	Scopes   []string  `json:"scopes"`
}

func (e *ResourceScopedTokenIssued) EventType() string     { return "ResourceScopedTokenIssued" }
func (e *ResourceScopedTokenIssued) OccurredAt() time.Time { return e.At }

// ResourceAudienceRejected は RFC 8707 — resource indicator が未登録・Disabled・複数指定
// のため fail-closed で拒否した。Authorize/PAR/Token(authorization_code) 経路が
// 対象。token-exchange の拒否は TokenExchangeRejected.reason で表現する。
type ResourceAudienceRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Reason   string    `json:"reason"`
}

func (e *ResourceAudienceRejected) EventType() string     { return "ResourceAudienceRejected" }
func (e *ResourceAudienceRejected) OccurredAt() time.Time { return e.At }

type RefreshTokenReuseDetected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	FamilyID string    `json:"familyId"`
	TokenID  string    `json:"tokenId"`
	ClientID string    `json:"clientId"`
	// UserID は再利用されたトークンの主体で、User の失効の入力になる。
	UserID string `json:"userId,omitempty"`
}

func (e *RefreshTokenReuseDetected) EventType() string     { return "RefreshTokenReuseDetected" }
func (e *RefreshTokenReuseDetected) OccurredAt() time.Time { return e.At }

type PARStored struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	RequestURI string    `json:"requestUri"`
	ClientID   string    `json:"clientId"`
}

func (e *PARStored) EventType() string     { return "PARStored" }
func (e *PARStored) OccurredAt() time.Time { return e.At }

type DeviceAuthorizationRequested struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Scopes   []string  `json:"scopes"`
}

func (e *DeviceAuthorizationRequested) EventType() string     { return "DeviceAuthorizationRequested" }
func (e *DeviceAuthorizationRequested) OccurredAt() time.Time { return e.At }

// BackchannelAuth events audit CIBA approval requests by UUID and never expose auth_req_id.

type BackchannelAuthRequested struct {
	At                time.Time `json:"-"`
	TenantID          string    `json:"tenantId"`
	ApprovalRequestID string    `json:"approvalRequestId"`
	ClientID          string    `json:"clientId"`
	AgentID           string    `json:"agentId,omitempty"`
	UserID            string    `json:"userId"`
	Scopes            []string  `json:"scopes"`
}

func (e *BackchannelAuthRequested) EventType() string     { return "BackchannelAuthRequested" }
func (e *BackchannelAuthRequested) OccurredAt() time.Time { return e.At }

type BackchannelAuthApproved struct {
	At                time.Time `json:"-"`
	TenantID          string    `json:"tenantId"`
	ApprovalRequestID string    `json:"approvalRequestId"`
	ClientID          string    `json:"clientId"`
	// AgentID は承認の対象となった Agent。Supervised な Agent が得たトークンを、
	// それを許可した人間へ結び付ける唯一の手掛かりになる (REQ-OAUTH2-050)。
	AgentID string `json:"agentId,omitempty"`
	UserID  string `json:"userId"`
}

func (e *BackchannelAuthApproved) EventType() string     { return "BackchannelAuthApproved" }
func (e *BackchannelAuthApproved) OccurredAt() time.Time { return e.At }

type BackchannelAuthDenied struct {
	At                time.Time `json:"-"`
	TenantID          string    `json:"tenantId"`
	ApprovalRequestID string    `json:"approvalRequestId"`
	ClientID          string    `json:"clientId"`
	UserID            string    `json:"userId"`
}

func (e *BackchannelAuthDenied) EventType() string     { return "BackchannelAuthDenied" }
func (e *BackchannelAuthDenied) OccurredAt() time.Time { return e.At }

type BackchannelAuthExpired struct {
	At                time.Time `json:"-"`
	TenantID          string    `json:"tenantId"`
	ApprovalRequestID string    `json:"approvalRequestId"`
	ClientID          string    `json:"clientId"`
	UserID            string    `json:"userId"`
}

func (e *BackchannelAuthExpired) EventType() string     { return "BackchannelAuthExpired" }
func (e *BackchannelAuthExpired) OccurredAt() time.Time { return e.At }

type DeviceAuthorizationApproved struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *DeviceAuthorizationApproved) EventType() string     { return "DeviceAuthorizationApproved" }
func (e *DeviceAuthorizationApproved) OccurredAt() time.Time { return e.At }

type DeviceAuthorizationDenied struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
}

func (e *DeviceAuthorizationDenied) EventType() string     { return "DeviceAuthorizationDenied" }
func (e *DeviceAuthorizationDenied) OccurredAt() time.Time { return e.At }

// ResourceAccessEvaluated は /resource-access/evaluate の結果 (REQ-OAUTH2-062)。
// 資源の識別子は残さない。資源の相関は AccessControl の FgaCheckEvaluated が担う。
type ResourceAccessEvaluated struct {
	At         time.Time `json:"-"`
	TenantID   string    `json:"tenantId"`
	RSClientID string    `json:"rsClientId"`
	// 以下は検証したトークンと提示の証明から導いた主体と相関。未検証のトークンでは空のまま (REQ-OAUTH2-062)。
	UserID           string   `json:"userId,omitempty"`
	AgentID          string   `json:"agentId,omitempty"`
	ActorChain       []string `json:"actorChain,omitempty"`
	RequestID        string   `json:"requestId,omitempty"`
	ResourceServerID string   `json:"resourceServerId"`
	TokenID          string   `json:"tokenId"`
	ResourceType     string   `json:"resourceType"`
	Relation         string   `json:"relation"`
	Permitted        bool     `json:"permitted"`
	Reasons          []string `json:"reasons"`
	// 以下は承認した操作の相関 (REQ-OAUTH2-065)。引数と operation_key の平文は残さない。
	ApprovalRequestID string `json:"approvalRequestId,omitempty"`
	OperationDigest   string `json:"operationDigest,omitempty"`
	Replayed          bool   `json:"replayed,omitempty"`
}

func (e *ResourceAccessEvaluated) EventType() string     { return "ResourceAccessEvaluated" }
func (e *ResourceAccessEvaluated) OccurredAt() time.Time { return e.At }

// IdJagIssued は ID-JAG を発行した。本文と利用者のデータは載せない (REQ-OAUTH2-066)。
type IdJagIssued struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	ClientID  string    `json:"clientId"`
	UserID    string    `json:"userId"`
	Audience  string    `json:"audience"`
	Resources []string  `json:"resources"`
	Scopes    []string  `json:"scopes"`
	JTI       string    `json:"jti"`
}

func (e *IdJagIssued) EventType() string     { return "IdJagIssued" }
func (e *IdJagIssued) OccurredAt() time.Time { return e.At }

// IdJagRedeemed は ID-JAG をアクセストークンへ償還した (REQ-OAUTH2-070)。
type IdJagRedeemed struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	Issuer   string    `json:"issuer"`
	ClientID string    `json:"clientId"`
	UserID   string    `json:"userId"`
	JTI      string    `json:"jti"`
}

func (e *IdJagRedeemed) EventType() string     { return "IdJagRedeemed" }
func (e *IdJagRedeemed) OccurredAt() time.Time { return e.At }

// IdJagRejected は ID-JAG の発行か償還を拒否した (REQ-OAUTH2-067、REQ-OAUTH2-069)。
type IdJagRejected struct {
	At       time.Time `json:"-"`
	TenantID string    `json:"tenantId"`
	ClientID string    `json:"clientId"`
	Stage    string    `json:"stage"`
	Reason   string    `json:"reason"`
}

func (e *IdJagRejected) EventType() string     { return "IdJagRejected" }
func (e *IdJagRejected) OccurredAt() time.Time { return e.At }

// CrossAppAccessPolicyCreated はアプリ間アクセスの許可を登録した (REQ-OAUTH2-072)。
type CrossAppAccessPolicyCreated struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	PolicyID  string    `json:"policyId"`
	Direction string    `json:"direction"`
	ClientID  string    `json:"clientId"`
}

func (e *CrossAppAccessPolicyCreated) EventType() string     { return "CrossAppAccessPolicyCreated" }
func (e *CrossAppAccessPolicyCreated) OccurredAt() time.Time { return e.At }

// CrossAppAccessPolicyDeleted はアプリ間アクセスの許可を削除した (REQ-OAUTH2-072)。
type CrossAppAccessPolicyDeleted struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	PolicyID  string    `json:"policyId"`
	Direction string    `json:"direction"`
	ClientID  string    `json:"clientId"`
}

func (e *CrossAppAccessPolicyDeleted) EventType() string     { return "CrossAppAccessPolicyDeleted" }
func (e *CrossAppAccessPolicyDeleted) OccurredAt() time.Time { return e.At }
