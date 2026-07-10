package spec

// Authentication bounded context の双子定義。MFA factor とログインセッション / 要求。

import "time"

type MfaFactor struct {
	UserID     string        `json:"user_id"`
	Type       MfaFactorType `json:"type"`
	Secret     *string       `json:"secret,omitempty"`
	Label      *string       `json:"label,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	LastUsedAt *time.Time    `json:"last_used_at,omitempty"`
}

func (m MfaFactor) Validate() error {
	return validate(mfaFactorSchema, &m)
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

func (c WebAuthnCredential) Validate() error {
	return validate(webAuthnCredentialSchema, &c)
}

// RecoveryCode は TOTP / WebAuthn 喪失時の backup recovery code 1 件 (wi-26 / ADR-087)。
// 平文は保存せず CodeHash (SHA-256 hex) のみを持つ。ConsumedAt が非 nil なら使用済み。
type RecoveryCode struct {
	UserID      string     `json:"user_id"`
	CodeHash    string     `json:"code_hash"`
	GeneratedAt time.Time  `json:"generated_at"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
}

func (c RecoveryCode) Validate() error {
	return validate(recoveryCodeSchema, &c)
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

func (s LoginSession) Validate() error {
	return validate(loginSessionSchema, &s)
}

type LoginRequest struct {
	RequestID string `json:"request_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Csrf      string `json:"csrf"`
}

func (r LoginRequest) Validate() error {
	return validate(loginRequestSchema, &r)
}
