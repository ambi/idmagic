package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// DeviceAuthorization は RFC 8628 device authorization の状態を表す。
type DeviceAuthorization struct {
	DeviceCodeHash  string                   `json:"device_code_hash"`
	TenantID        string                   `json:"tenant_id"`
	UserCode        string                   `json:"user_code"`
	ClientID        string                   `json:"client_id"`
	Scopes          []string                 `json:"scopes"`
	State           spec.DeviceCodeFlowState `json:"state"`
	UserID          *string                  `json:"user_id,omitempty"`
	AuthTime        *int64                   `json:"auth_time,omitempty"`
	IntervalSeconds int                      `json:"interval_seconds"`
	LastPolledAt    *time.Time               `json:"last_polled_at,omitempty"`
	IssuedFamilyID  *string                  `json:"issued_family_id,omitempty"`
	IssuedAt        time.Time                `json:"issued_at"`
	ExpiresAt       time.Time                `json:"expires_at"`
}

func (d DeviceAuthorization) Validate() error { return spec.ValidateDeviceAuthorization(&d) }
