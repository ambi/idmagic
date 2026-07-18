package eventsink

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/logging"
)

// LogPublisher は broker を用意せず outbox を drain したいローカル/オンプレの
// 最小構成向け Publisher。イベントをログ出力して ack するだけで、外部ストリームへは
// 送らない。消費側が必要になったら kafka / pubsub sink へ切り替える ([[ADR-120]])。
type LogPublisher struct{}

func NewLogPublisher() *LogPublisher { return &LogPublisher{} }

func (p *LogPublisher) Name() string { return "log" }

func (p *LogPublisher) Close() {}

func (p *LogPublisher) Publish(ctx context.Context, m OutboxMessage) error {
	logging.Info(ctx, "outbox event relayed",
		"topic", m.Topic,
		"event_type", m.EventType,
		"outbox_id", m.OutboxID,
		"key", m.Key,
	)
	return nil
}
