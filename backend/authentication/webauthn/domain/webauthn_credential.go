package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"

	z "github.com/Oudwins/zog"
)

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
