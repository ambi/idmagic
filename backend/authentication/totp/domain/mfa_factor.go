package domain

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
