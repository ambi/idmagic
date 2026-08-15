package ports

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// TrustedDeviceRepository は信頼済みデバイスの永続化 (wi-91)。cookie の平文は保存せず、
// selector と verifier_hash だけを持つ。失効は行を削除せず tombstone として残す。
type TrustedDeviceRepository interface {
	// FindBySelector は selector で 1 件を引く。テナント境界は呼び出し側ではなく
	// この照合で閉じるため、tenantID が一致しない行は「無い」として扱う。
	FindBySelector(ctx context.Context, tenantID, selector string) (*domain.TrustedDevice, error)
	// FindByID は本人のデバイス 1 件を失効済みも含めて引く。tenantID / userID が
	// 一致しない行は「無い」として扱うので、他人のデバイス ID の存在は試せない。
	FindByID(ctx context.Context, tenantID, userID, deviceID string) (*domain.TrustedDevice, error)
	// ListActiveByUser は失効しておらず絶対期限内の行を last_used_at の降順で返す。
	// idle 期限の判定は時刻の比較なので呼び出し側 (domain.Active) が行う。
	ListActiveByUser(ctx context.Context, tenantID, userID string) ([]*domain.TrustedDevice, error)
	Save(ctx context.Context, device *domain.TrustedDevice) error
	// RevokeAllForUser は対象ユーザーの未失効の行をすべて失効させ、失効した行を返す。
	// 既に失効済みの行は idempotent にスキップする。
	RevokeAllForUser(
		ctx context.Context, tenantID, userID string,
		reason spec.TrustedDeviceRevokeReason, now time.Time,
	) ([]*domain.TrustedDevice, error)
	// DeleteAllForSub は匿名化 cascade から呼ばれる物理削除。
	DeleteAllForSub(ctx context.Context, userID string) error
}
