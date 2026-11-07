package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// WebAuthnCredential は登録済みの WebAuthn / Passkey credential 1 件 (wi-26)。
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
	return spec.All(
		// CredentialID は主キーそのもの。WebAuthn は credential ID を 1023 バイト
		// 以下と定めるので base64url で 1364 文字になり、上限は余白を残した数である。
		spec.Required("CredentialID", c.CredentialID,
			spec.KeyString(spec.LengthWebAuthnCredentialID, spec.BytesWebAuthnCredentialID)),
		spec.Required("UserID", c.UserID),
		spec.Required("PublicKey", c.PublicKey),
		spec.Required("CreatedAt", c.CreatedAt),
	)
}
