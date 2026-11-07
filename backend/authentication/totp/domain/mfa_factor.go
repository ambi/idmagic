package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type MfaFactor struct {
	UserID     string             `json:"user_id"`
	Type       spec.MfaFactorType `json:"type"`
	Secret     *string            `json:"secret,omitempty"`
	Label      *string            `json:"label,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	LastUsedAt *time.Time         `json:"last_used_at,omitempty"`
}

func (m MfaFactor) Validate() error {
	return spec.All(
		spec.Required("UserID", m.UserID),
		spec.Required("Type", m.Type, spec.Must(spec.MfaFactorType.Valid, "mfa factor type is not in enum")),
		spec.Required("CreatedAt", m.CreatedAt),
		spec.Condition(m.Type != spec.MfaFactorTOTP || m.Secret != nil && *m.Secret != "",
			"totp factor requires secret"),
	)
}
