package domain

// Authentication bounded context の業務型。MFA factor とログインセッション / 要求。

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"

	z "github.com/Oudwins/zog"
)

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
	ID                    string    `json:"id"`
	TenantID              string    `json:"tenant_id"`
	UserID                string    `json:"user_id"`
	AuthTime              int64     `json:"auth_time"`
	AMR                   []string  `json:"amr"`
	ACR                   string    `json:"acr"`
	AuthenticationPending bool      `json:"authentication_pending"`
	ExpiresAt             time.Time `json:"expires_at"`
	// StepUpAt は直近で password / MFA による step-up 再認証が成立した時刻 (Unix 秒、
	// 未実施は 0)。高 sensitivity な self-service 操作の recency 判定に使う (ADR-043)。
	StepUpAt int64 `json:"step_up_at,omitempty"`
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
