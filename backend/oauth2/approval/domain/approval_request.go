// Package domain owns approval requests that hold a client request until a
// human decision is available. CIBA bookkeeping is co-located with the
// decision record so state transitions and polling updates stay atomic.
package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	// DefaultTTL gives a user enough time to act on another device without
	// leaving undecided requests around for long.
	DefaultTTL = 300 * time.Second
	// MaxTTL is the largest requested_expiry accepted from a client.
	MaxTTL = 600 * time.Second
	// MaxBindingMessageLength keeps the identifier short enough for safe display.
	MaxBindingMessageLength = 64
)

// ApprovalRequest holds the human decision for one requested action.
type ApprovalRequest struct {
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

	// The following fields are transport bookkeeping and do not alter the
	// meaning of the human decision.

	// AuthReqIDHash is the SHA-256 digest of the bearer secret.
	AuthReqIDHash string `json:"auth_req_id_hash"`
	// IntervalSeconds is the current minimum polling interval.
	IntervalSeconds int `json:"interval_seconds"`
	// LastPolledAt records the latest pending-state poll.
	LastPolledAt *time.Time `json:"last_polled_at,omitempty"`

	RequestedAt time.Time  `json:"requested_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
}

func (a ApprovalRequest) Validate() error { return spec.ValidateApprovalRequest(&a) }

// NewApprovalRequestID returns the stable identifier exposed to the account portal.
func NewApprovalRequestID() (string, error) { return spec.NewUUIDv4() }

// GenerateAuthReqID returns a 256-bit bearer secret suitable for auth_req_id.
func GenerateAuthReqID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
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
	if now.IsZero() {
		now = time.Now().UTC()
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
