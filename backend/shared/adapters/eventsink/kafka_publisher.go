package eventsink

import (
	"context"
	"strconv"

	"github.com/twmb/franz-go/pkg/kgo"
)

// KafkaPublisher は outbox メッセージを Kafka (wire 互換の Redpanda を含む) へ送る
// Publisher 実装。ローカル/オンプレの自ホスト broker でもクラウドのマネージド
// broker でもそのまま使える (default sink)。
type KafkaPublisher struct {
	client *kgo.Client
}

// NewKafkaPublisher は seed broker と clientID から Kafka producer を構築する。
func NewKafkaPublisher(brokers []string, clientID string) (*KafkaPublisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID(clientID),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),
	)
	if err != nil {
		return nil, err
	}
	return &KafkaPublisher{client: client}, nil
}

func (p *KafkaPublisher) Name() string { return "kafka" }

func (p *KafkaPublisher) Close() { p.client.Close() }

func (p *KafkaPublisher) Publish(ctx context.Context, m OutboxMessage) error {
	record := &kgo.Record{
		Topic: m.Topic,
		Key:   []byte(m.Key),
		Value: m.Payload,
		Headers: []kgo.RecordHeader{
			{Key: "event_type", Value: []byte(m.EventType)},
			{Key: "outbox_id", Value: []byte(strconv.FormatInt(m.OutboxID, 10))},
		},
	}
	return p.client.ProduceSync(ctx, record).FirstErr()
}
