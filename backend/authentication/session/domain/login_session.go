package domain

import (
	"time"

	authndomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/shared/spec"

	z "github.com/Oudwins/zog"
)

// LoginPendingPurpose は AuthenticationContext.PendingPurpose と型を共有するため、
// context ルートの共有 domain に定義がある (authentication_context.go)。
type LoginPendingPurpose = authndomain.LoginPendingPurpose

const (
	LoginPendingNone       = authndomain.LoginPendingNone
	LoginPendingChallenge  = authndomain.LoginPendingChallenge
	LoginPendingEnrollment = authndomain.LoginPendingEnrollment
)

type LoginSession struct {
	ID                    string              `json:"id"`
	TenantID              string              `json:"tenant_id"`
	UserID                string              `json:"user_id"`
	AuthTime              int64               `json:"auth_time"`
	AMR                   []string            `json:"amr"`
	ACR                   string              `json:"acr"`
	AuthenticationPending bool                `json:"authentication_pending"`
	PendingPurpose        LoginPendingPurpose `json:"pending_purpose,omitempty"`
	EnrollmentDeadline    *time.Time          `json:"enrollment_deadline,omitempty"`
	EnrollmentBypassID    string              `json:"enrollment_bypass_id,omitempty"`
	ExpiresAt             time.Time           `json:"expires_at"`
	// StepUpAt は直近で password / MFA による step-up 再認証が成立した時刻 (Unix 秒、
	// 未実施は 0)。高 sensitivity な self-service 操作の recency 判定に使う (ADR-043)。
	StepUpAt int64 `json:"step_up_at,omitempty"`
	// LastSeenAt はセッションが認証結果として意味上利用された直近時刻。
	// LoginSessionTouchInterval 未満の再 touch では更新しない粗粒度な値で、
	// idle timeout 判定はこの粒度で近似する (wi-253)。
	LastSeenAt time.Time `json:"last_seen_at,omitzero"`
	// RevokedAt / RevokeReason は失効の tombstone。物理削除せず残し、認証解決は
	// RevokedAt != nil を fail-closed で無効とみなす。再失効は最初の値を保持する
	// idempotent 操作 (ADR-126)。
	RevokedAt    *time.Time             `json:"revoked_at,omitempty"`
	RevokeReason *spec.SessionEndReason `json:"revoke_reason,omitempty"`
}

// LoginSessionTouchInterval は last_seen_at を実際に書き換える最小間隔。この間隔未満の
// touch は永続化しない粗粒度な近似とし、書き込み量を抑える (wi-253 Plan §4)。
const LoginSessionTouchInterval = 5 * time.Minute

// Active は認証解決が LoginSession を有効とみなせるかを fail-closed に判定する。
// revoked または期限切れは無効。
func (s LoginSession) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}

// Revoke はセッションを tombstone として失効させる。最初の失効だけが revoked_at /
// revoke_reason を確定し、以降の呼び出しは idempotent な no-op になる
// (ADR-126、self/admin どちらの失効経路からも安全に再送できる)。
func (s *LoginSession) Revoke(reason spec.SessionEndReason, now time.Time) {
	if s.RevokedAt != nil {
		return
	}
	t := now
	s.RevokedAt = &t
	s.RevokeReason = &reason
}

// Touch は last_seen_at を LoginSessionTouchInterval の粒度で更新する。実際に更新した
// 場合だけ true を返す。呼び出し側 (adapter) はこれを使って書き込みの要否を判定できる。
func (s *LoginSession) Touch(now time.Time) bool {
	if !s.LastSeenAt.IsZero() && now.Sub(s.LastSeenAt) < LoginSessionTouchInterval {
		return false
	}
	s.LastSeenAt = now
	return true
}

var loginSessionSchema = z.Struct(z.Shape{
	"ID":        z.String().UUID().Required(),
	"UserID":    z.String().Required(),
	"AMR":       z.Slice(z.String()).Min(1).Required(),
	"ACR":       z.String().Required(),
	"ExpiresAt": z.Time().Required(),
}).TestFunc(func(value any, _ z.Ctx) bool {
	sess, ok := value.(*LoginSession)
	return ok && (sess.RevokedAt == nil) == (sess.RevokeReason == nil)
}, z.Message("revoked_at and revoke_reason must be set together"))

func (s LoginSession) Validate() error {
	return spec.Validate(loginSessionSchema, &s)
}

type LoginRequest struct {
	RequestID string `json:"request_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Csrf      string `json:"csrf"`
}

var loginRequestSchema = z.Struct(z.Shape{
	"RequestID": z.String().UUID().Required(),
	"Username":  z.String().Required(),
	"Password":  z.String().Required(),
})

func (r LoginRequest) Validate() error {
	return spec.Validate(loginRequestSchema, &r)
}
