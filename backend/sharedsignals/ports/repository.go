// Package ports は SharedSignals モジュールの repository/adapter インターフェースを所有する。
package ports

import (
	"context"
	"time"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// AgentRevocationEpochRepository は Agent ごとの revocation epoch を永続化する。
// Agent が未失効の場合 FindByAgent は (nil, nil) を返す。
type AgentRevocationEpochRepository interface {
	FindByAgent(ctx context.Context, tenantID, agentID string) (*ssdomain.AgentRevocationEpoch, error)
	// Advance は epoch を fail-closed に前進させる。既存 epoch が存在し、それより後の
	// 時刻でなければ ErrEpochNotAdvancing を返し既存値を保持する (単調増加保証)。
	Advance(ctx context.Context, epoch ssdomain.AgentRevocationEpoch) error
}

// UserRevocationEpochRepository は User ごとの revocation epoch を永続化する。
// User が未失効の場合 FindByUser は (nil, nil) を返す。
type UserRevocationEpochRepository interface {
	FindByUser(ctx context.Context, tenantID, userID string) (*ssdomain.UserRevocationEpoch, error)
	// Advance は AgentRevocationEpochRepository.Advance と同じ単調増加を保証する。
	Advance(ctx context.Context, epoch ssdomain.UserRevocationEpoch) error
}

// SsfStreamRepository は SsfStream を永続化する。
type SsfStreamRepository interface {
	ListAll(ctx context.Context, tenantID string) ([]*ssdomain.SsfStream, error)
	FindByID(ctx context.Context, tenantID, id string) (*ssdomain.SsfStream, error)
	Save(ctx context.Context, stream *ssdomain.SsfStream) error
	Delete(ctx context.Context, tenantID, id string) error
	// RecordDeliveryFailure は連続の失敗を数え、閾値で enabled から disabled にしたときに true を返す。
	RecordDeliveryFailure(ctx context.Context, tenantID, id string, now time.Time) (bool, error)
	// RecordDeliverySuccess は連続の失敗の記録を消す。
	RecordDeliverySuccess(ctx context.Context, tenantID, id string) error
}

// SsfTransmitterConfigRepository は SsfTransmitterConfig を永続化する (stream_id に 1 対 1)。
type SsfTransmitterConfigRepository interface {
	FindByStream(ctx context.Context, tenantID, streamID string) (*ssdomain.SsfTransmitterConfig, error)
	Save(ctx context.Context, tenantID string, config *ssdomain.SsfTransmitterConfig) error
	Delete(ctx context.Context, tenantID, streamID string) error
}

// SsfReceiverConfigRepository は SsfReceiverConfig を永続化する (stream_id に 1 対 1)。
type SsfReceiverConfigRepository interface {
	FindByStream(ctx context.Context, tenantID, streamID string) (*ssdomain.SsfReceiverConfig, error)
	Save(ctx context.Context, tenantID string, config *ssdomain.SsfReceiverConfig) error
	Delete(ctx context.Context, tenantID, streamID string) error
}

// SecurityEventDeliveryRepository は outbound SET 配送 (outbox) を永続化する。
type SecurityEventDeliveryRepository interface {
	ListByStream(ctx context.Context, tenantID, streamID string) ([]*ssdomain.SecurityEventDelivery, error)
	// ClaimDue は next_attempt_at <= now の pending/failed 配送を、ストリームごとに perStream 件、
	// 全体で limit 件まで取得し、next_attempt_at を leaseUntil へ進める。並行した呼び出しは同じ行を返さない。
	ClaimDue(ctx context.Context, now, leaseUntil time.Time, perStream, limit int) ([]*ssdomain.SecurityEventDelivery, error)
	Save(ctx context.Context, delivery *ssdomain.SecurityEventDelivery) error
	// PurgeSettled は before より前に作られた delivered と dead_letter の配送を消す。
	PurgeSettled(ctx context.Context, before time.Time) (int64, error)
	// RedeliverDeadLettered は stream の dead_letter の配送を不可分に pending へ戻し、
	// 戻した配送を返す。並行した要求でも、各配送を戻すのは一方だけである。
	RedeliverDeadLettered(ctx context.Context, tenantID, streamID string) ([]*ssdomain.SecurityEventDelivery, error)
}

// ReceivedSecurityEventRepository は inbound SET の受理記録を永続化し、jti 重複を検知する。
type ReceivedSecurityEventRepository interface {
	// ExistsByJTI は同一 stream 内に同じ set_jti の受理記録が既にあるかを返す (replay 検知)。
	ExistsByJTI(ctx context.Context, tenantID, streamID, setJTI string) (bool, error)
	Save(ctx context.Context, event *ssdomain.ReceivedSecurityEvent) error
}
