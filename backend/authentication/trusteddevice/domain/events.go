package domain

import (
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// TrustedDeviceRegistered はログインで本物の第二要素が成立した直後に、本人の明示同意で
// デバイスを記憶したことを表す (wi-91)。cookie の平文も User-Agent も IP も載せない。
type TrustedDeviceRegistered struct {
	At        time.Time `json:"-"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	DeviceID  string    `json:"deviceId"`
	Factor    string    `json:"factor"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (e *TrustedDeviceRegistered) EventType() string     { return "TrustedDeviceRegistered" }
func (e *TrustedDeviceRegistered) OccurredAt() time.Time { return e.At }

// TrustedDeviceRevoked は信頼済みデバイスを失効したことを表す。本人の明示的な失効に加え、
// 資格情報の変更・認証器のリセット・アカウント無効化・全セッション失効による一括失効でも発行する。
type TrustedDeviceRevoked struct {
	At       time.Time                      `json:"-"`
	TenantID string                         `json:"tenantId"`
	UserID   string                         `json:"userId"`
	DeviceID string                         `json:"deviceId"`
	Reason   spec.TrustedDeviceRevokeReason `json:"reason"`
}

func (e *TrustedDeviceRevoked) EventType() string     { return "TrustedDeviceRevoked" }
func (e *TrustedDeviceRevoked) OccurredAt() time.Time { return e.At }
