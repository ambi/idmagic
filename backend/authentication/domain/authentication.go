package domain

// Authentication bounded context の業務型。MFA factor とログインセッション / 要求。

import (
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"

	z "github.com/Oudwins/zog"
)

var ErrInvalidMfaEnrollmentBypass = errors.New("invalid MFA enrollment bypass")

type MfaFactor struct {
	UserID     string             `json:"user_id"`
	Type       spec.MfaFactorType `json:"type"`
	Secret     *string            `json:"secret,omitempty"`
	Label      *string            `json:"label,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	LastUsedAt *time.Time         `json:"last_used_at,omitempty"`
}

var mfaFactorSchema = z.Struct(z.Shape{
	"UserID": z.String().Required(),
	"Type": z.StringLike[spec.MfaFactorType]().TestFunc(
		func(value *spec.MfaFactorType, _ z.Ctx) bool { return value.Valid() },
		z.Message("mfa factor type is not in enum"),
	).Required(),
	"CreatedAt": z.Time().Required(),
}).TestFunc(func(value any, _ z.Ctx) bool {
	factor, ok := value.(*MfaFactor)
	return ok && (factor.Type != spec.MfaFactorTOTP ||
		factor.Secret != nil && *factor.Secret != "")
}, z.Message("totp factor requires secret"))

func (m MfaFactor) Validate() error {
	return spec.Validate(mfaFactorSchema, &m)
}

// WebAuthnCredential は登録済みの WebAuthn / Passkey credential 1 件 (wi-26 / ADR-087)。
// 1 ユーザーが複数持てるため MfaFactor とは別集合とし、credential_id で一意識別する。
// PublicKey は COSE 公開鍵 (base64url)、SignCount は clone 検出用の署名カウンタ。
type WebAuthnCredential struct {
	CredentialID   string     `json:"credential_id"`
	UserID         string     `json:"user_id"`
	PublicKey      string     `json:"public_key"`
	SignCount      uint32     `json:"sign_count"`
	Transports     []string   `json:"transports"`
	AAGUID         *string    `json:"aaguid,omitempty"`
	Label          *string    `json:"label,omitempty"`
	BackupEligible bool       `json:"backup_eligible"`
	BackupState    bool       `json:"backup_state"`
	CreatedAt      time.Time  `json:"created_at"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
}

var webAuthnCredentialSchema = z.Struct(z.Shape{
	"CredentialID": z.String().Required(),
	"UserID":       z.String().Required(),
	"PublicKey":    z.String().Required(),
	"CreatedAt":    z.Time().Required(),
})

func (c WebAuthnCredential) Validate() error {
	return spec.Validate(webAuthnCredentialSchema, &c)
}

// RecoveryCode は TOTP / WebAuthn 喪失時の backup recovery code 1 件 (wi-26 / ADR-087)。
// 平文は保存せず CodeHash (SHA-256 hex) のみを持つ。ConsumedAt が非 nil なら使用済み。
type RecoveryCode struct {
	UserID      string     `json:"user_id"`
	CodeHash    string     `json:"code_hash"`
	GeneratedAt time.Time  `json:"generated_at"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
}

var recoveryCodeSchema = z.Struct(z.Shape{
	"UserID":      z.String().Required(),
	"CodeHash":    z.String().Required(),
	"GeneratedAt": z.Time().Required(),
})

func (c RecoveryCode) Validate() error {
	return spec.Validate(recoveryCodeSchema, &c)
}

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
}

type LoginPendingPurpose string

const (
	LoginPendingNone       LoginPendingPurpose = "None"
	LoginPendingChallenge  LoginPendingPurpose = "Challenge"
	LoginPendingEnrollment LoginPendingPurpose = "Enrollment"
)

func (p LoginPendingPurpose) Valid() bool {
	switch p {
	case LoginPendingNone, LoginPendingChallenge, LoginPendingEnrollment:
		return true
	}
	return false
}

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

var loginSessionSchema = z.Struct(z.Shape{
	"ID":        z.String().UUID().Required(),
	"UserID":    z.String().Required(),
	"AMR":       z.Slice(z.String()).Min(1).Required(),
	"ACR":       z.String().Required(),
	"ExpiresAt": z.Time().Required(),
})

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
