package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"

	z "github.com/Oudwins/zog"
)

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
