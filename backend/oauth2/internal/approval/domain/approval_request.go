// Package domain owns approval requests that hold a client request until a
// human decision is available. CIBA bookkeeping is co-located with the
// decision record so state transitions and polling updates stay atomic.
package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/ambi/idmagic/backend/shared/security/entropy"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	// DefaultTTL gives a user enough time to act on another device without
	// leaving undecided requests around for long.
	DefaultTTL = 300 * time.Second
	// MaxTTL is the largest requested_expiry accepted from a client.
	MaxTTL = 600 * time.Second
	// MaxBindingMessageLength keeps the identifier short enough for safe display.
	// Counted in Unicode code points, like every other string limit.
	MaxBindingMessageLength = spec.LengthHandle
)

// ApprovalInitiator は承認リクエストを起票した側である。起票の経路ごとに、交換と参照の可否が変わる。
type ApprovalInitiator string

const (
	// InitiatedByClientBackchannel は CIBA で要求したクライアントである。保存済みの記録の空の値も、これを表す。
	InitiatedByClientBackchannel ApprovalInitiator = "client_backchannel"
	// InitiatedByResourceServer は評価で起票したリソースサーバーである (REQ-OAUTH2-093)。
	// auth_req_id を持たず、トークンへ交換できず、評価が識別子で参照する。
	InitiatedByResourceServer ApprovalInitiator = "resource_server"
)

// ResourceServerApprovalTTL は評価が起票した承認の有効期間である。本人がメールかクライアントの URL から
// ポータルへ移り、ステップアップ認証を経て判断し、クライアントが再送するまでを含めるため、CIBA の上限にそろえる。
const ResourceServerApprovalTTL = MaxTTL

// MaxPendingResourceServerApprovals は、リソースサーバーが起票した保留中の承認の、利用者ごとの上限である。
const MaxPendingResourceServerApprovals = 10

// ApprovalRequest holds the human decision for one requested action.
type ApprovalRequest struct {
	ToolName string `json:"tool_name,omitempty"`
	ToolID   string `json:"tool_id,omitempty"`
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	ClientID string `json:"client_id"`
	// AgentID identifies the agent bound to the client, when one exists.
	AgentID *string `json:"agent_id,omitempty"`
	// UserID is the only subject allowed to decide this request.
	UserID               string                     `json:"user_id"`
	Scopes               []string                   `json:"scopes"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
	BindingMessage       *string                    `json:"binding_message,omitempty"`
	State                spec.ApprovalRequestState  `json:"state"`

	// Initiator は起票した側で、空は CIBA を表す。
	Initiator ApprovalInitiator `json:"initiator,omitempty"`
	// ResourceServerID は、評価で起票した登録である。参照できる評価をこの登録に限る。
	ResourceServerID *string `json:"resource_server_id,omitempty"`
	// OperationDigest は、評価で起票した操作と引数の digest で、同じ操作の重複起票を見分ける。
	OperationDigest *string `json:"operation_digest,omitempty"`

	// The following fields are transport bookkeeping and do not alter the
	// meaning of the human decision. Only CIBA requests carry them.

	// AuthReqIDHash is the SHA-256 digest of the bearer secret.
	AuthReqIDHash string `json:"auth_req_id_hash,omitempty"`
	// IntervalSeconds is the current minimum polling interval.
	IntervalSeconds int `json:"interval_seconds"`
	// LastPolledAt records the latest pending-state poll.
	LastPolledAt *time.Time `json:"last_polled_at,omitempty"`

	RequestedAt time.Time  `json:"requested_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
	// Use is the single right to execute the approved operation (REQ-OAUTH2-065).
	Use *ApprovalUse `json:"use,omitempty"`
}

func (a ApprovalRequest) Validate() error {
	return spec.All(
		spec.Required("ID", a.ID, spec.UUID()),
		spec.Required("ClientID", a.ClientID),
		spec.Required("UserID", a.UserID),
		// CIBA の binding_message。start 時の検査 (approval/usecases) も同じ
		// コードポイント数で数える。
		spec.OptionalPtr("BindingMessage", a.BindingMessage, spec.CharsAtMost(spec.LengthHandle)),
		spec.Required("State", a.State, spec.Must(spec.ApprovalRequestState.Valid, "state is not in enum")),
		spec.Required("RequestedAt", a.RequestedAt),
		spec.Required("ExpiresAt", a.ExpiresAt),
		a.validateInitiator(),
	)
}

// validateInitiator は起票の経路ごとの項目を確かめる。評価の起票が auth_req_id を持つと、
// トークンへ交換する経路ができる。
func (a ApprovalRequest) validateInitiator() spec.Issues {
	switch a.Initiator {
	case "", InitiatedByClientBackchannel:
		return append(spec.Required("AuthReqIDHash", a.AuthReqIDHash),
			spec.Required("IntervalSeconds", a.IntervalSeconds, spec.GreaterThan(0))...)
	case InitiatedByResourceServer:
		issues := spec.Condition(a.ResourceServerID != nil && *a.ResourceServerID != "", "resource_server_id is required")
		issues = append(issues, spec.Condition(a.OperationDigest != nil && *a.OperationDigest != "", "operation_digest is required")...)
		return append(issues, spec.Condition(a.AuthReqIDHash == "" && a.IntervalSeconds == 0,
			"a resource server request has no auth_req_id or polling interval")...)
	default:
		return spec.Condition(false, "initiator is not in enum")
	}
}

// InitiatedByResourceServer は、評価が起票した承認リクエストかを返す。
func (a ApprovalRequest) InitiatedByResourceServer() bool {
	return a.Initiator == InitiatedByResourceServer
}

// NewApprovalRequestID returns the stable identifier exposed to the account portal.
func NewApprovalRequestID() (string, error) { return spec.NewUUIDv4() }

// GenerateAuthReqID returns a 256-bit bearer secret suitable for auth_req_id.
func GenerateAuthReqID(random entropy.Source) (string, error) {
	b := make([]byte, 32)
	if err := random.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func HashAuthReqID(authReqID string) string {
	sum := sha256.Sum256([]byte(authReqID))
	return hex.EncodeToString(sum[:])
}

// ResolveTTL applies the default to an omitted requested_expiry. Callers must
// reject non-positive or over-limit values before invoking it.
func ResolveTTL(requestedExpirySeconds *int) time.Duration {
	if requestedExpirySeconds == nil {
		return DefaultTTL
	}
	return time.Duration(*requestedExpirySeconds) * time.Second
}

// IsExpired reports whether the request is terminally expired or past its deadline.
func IsExpired(rec *ApprovalRequest, now time.Time) bool {
	if rec.State == spec.ApprovalExpired {
		return true
	}
	return !now.Before(rec.ExpiresAt)
}

// IsPollTooFast reports whether a poll violates the current minimum interval.
func IsPollTooFast(rec *ApprovalRequest, now time.Time) bool {
	if rec.LastPolledAt == nil {
		return false
	}
	return now.Sub(*rec.LastPolledAt) < time.Duration(rec.IntervalSeconds)*time.Second
}
