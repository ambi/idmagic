package domain

import (
	"errors"
	"time"
)

var ErrInvalidMfaEnrollmentBypass = errors.New("invalid MFA enrollment bypass")

type MfaEnrollmentBypass struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id"`
	IssuedBy   string     `json:"issued_by"`
	IssuedAt   time.Time  `json:"issued_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ExpiredAt  *time.Time `json:"expired_at,omitempty"`
}

func (b MfaEnrollmentBypass) Available(now time.Time) bool {
	return b.ConsumedAt == nil && b.RevokedAt == nil && b.ExpiredAt == nil && now.Before(b.ExpiresAt)
}

func (b MfaEnrollmentBypass) Validate() error {
	if b.ID == "" || b.TenantID == "" || b.UserID == "" || b.IssuedBy == "" ||
		b.IssuedAt.IsZero() || !b.ExpiresAt.After(b.IssuedAt) ||
		boolCount(b.ConsumedAt != nil, b.RevokedAt != nil, b.ExpiredAt != nil) > 1 {
		return ErrInvalidMfaEnrollmentBypass
	}
	return nil
}

func boolCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}
	return count
}

type MfaEnrollmentDecision string

const (
	MfaEnrollmentNotRequired MfaEnrollmentDecision = "not_required"
	MfaEnrollmentRequired    MfaEnrollmentDecision = "enrollment_required"
	MfaEnrollmentDenied      MfaEnrollmentDecision = "denied"
)

// EvaluateMfaEnrollment は MFA 必須ポリシー下の未登録 user を fail-closed で判定する。
// 強制開始前は通常ログイン、開始後は policy の猶予内かつ有効な管理者 bypass がある場合だけ登録へ進む。
func EvaluateMfaEnrollment(now time.Time, enforcementStart *time.Time, grace time.Duration, allowBypass bool, bypass *MfaEnrollmentBypass) (MfaEnrollmentDecision, *time.Time) {
	if enforcementStart == nil || now.Before(*enforcementStart) {
		return MfaEnrollmentNotRequired, nil
	}
	deadline := enforcementStart.Add(grace)
	if grace <= 0 || now.After(deadline) || !allowBypass || bypass == nil || !bypass.Available(now) {
		return MfaEnrollmentDenied, &deadline
	}
	return MfaEnrollmentRequired, &deadline
}
