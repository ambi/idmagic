package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// RecoveryCode は TOTP / WebAuthn 喪失時の backup recovery code 1 件 (wi-26)。
// 平文は保存せず CodeHash (SHA-256 hex) のみを持つ。ConsumedAt が非 nil なら使用済み。
type RecoveryCode struct {
	UserID      string     `json:"user_id"`
	CodeHash    string     `json:"code_hash"`
	GeneratedAt time.Time  `json:"generated_at"`
	ConsumedAt  *time.Time `json:"consumed_at,omitempty"`
}

func (c RecoveryCode) Validate() error {
	return spec.All(
		spec.Required("UserID", c.UserID),
		spec.Required("CodeHash", c.CodeHash),
		spec.Required("GeneratedAt", c.GeneratedAt),
	)
}
