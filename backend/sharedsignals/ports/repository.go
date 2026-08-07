// Package ports は SharedSignals bounded context の repository/adapter インターフェースを所有する。
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
	// Advance は epoch を fail-closed に前進させる。既存 epoch が存在し、それ以降の
	// 時刻でなければ ErrEpochNotAdvancing を返し既存値を保持する (単調増加保証)。
	Advance(ctx context.Context, epoch ssdomain.AgentRevocationEpoch) error
}

// SsfStreamRepository は SsfStream を永続化する。
type SsfStreamRepository interface {
	ListByTenant(ctx context.Context, tenantID string) ([]*ssdomain.SsfStream, error)
	FindByID(ctx context.Context, tenantID, id string) (*ssdomain.SsfStream, error)
	Save(ctx context.Context, stream *ssdomain.SsfStream) error
	Delete(ctx context.Context, tenantID, id string) error
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
	// ListDue は next_attempt_at <= now の pending/failed 配送を返す (projector/worker が消費する)。
	ListDue(ctx context.Context, now time.Time, limit int) ([]*ssdomain.SecurityEventDelivery, error)
	Save(ctx context.Context, delivery *ssdomain.SecurityEventDelivery) error
}

// ReceivedSecurityEventRepository は inbound SET の受理記録を永続化し、jti 重複を検知する。
type ReceivedSecurityEventRepository interface {
	// ExistsByJTI は同一 stream 内に同じ set_jti の受理記録が既にあるかを返す (replay 検知)。
	ExistsByJTI(ctx context.Context, tenantID, streamID, setJTI string) (bool, error)
	Save(ctx context.Context, event *ssdomain.ReceivedSecurityEvent) error
}
