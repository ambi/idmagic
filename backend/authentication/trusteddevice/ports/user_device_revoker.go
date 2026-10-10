package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// UserDeviceRevoker は利用者の信頼済みデバイスをすべて失効させ、発行すべき TrustedDeviceRevoked を
// 返す。発行は呼び出し側が自分の発行先で行う。アカウントの無効化のように、呼び出し側の
// トランザクションに結びついた発行先でイベントを出す必要があるからである。
type UserDeviceRevoker interface {
	RevokeAllForUser(ctx context.Context, tenantID, userID string, reason spec.TrustedDeviceRevokeReason, now time.Time) ([]spec.DomainEvent, error)
}
