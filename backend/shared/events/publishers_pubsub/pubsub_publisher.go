//go:build pubsub

// Package eventsink の Pub/Sub アダプタは build タグ `pubsub` 付きでのみコンパイルされる。
// 既定ビルド (ローカル/オンプレ/dev) は GCP SDK を一切含めない ([[ADR-120]])。
package publishers_pubsub

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"cloud.google.com/go/pubsub"
	eventports "github.com/ambi/idmagic/backend/shared/events/ports"
)

// PubSubPublisher は outbox メッセージを GCP Pub/Sub トピックへ送る Publisher 実装。
// per-aggregate ordering を保つため message ordering を有効化し、partition key を
// ordering key に割り当てる。
type PubSubPublisher struct {
	client *pubsub.Client
	mu     sync.Mutex
	topics map[string]*pubsub.Topic
}

// NewPubSubPublisher は projectID の Pub/Sub client を構築する。戻り値型は Publisher で、
// 非タグビルドの stub と同一シグネチャに揃えている。
func NewPubSubPublisher(ctx context.Context, projectID string) (eventports.Publisher, error) {
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &PubSubPublisher{client: client, topics: map[string]*pubsub.Topic{}}, nil
}

func (p *PubSubPublisher) Name() string { return "pubsub" }

func (p *PubSubPublisher) topic(name string) *pubsub.Topic {
	p.mu.Lock()
	defer p.mu.Unlock()
	t, ok := p.topics[name]
	if !ok {
		t = p.client.Topic(name)
		t.EnableMessageOrdering = true
		p.topics[name] = t
	}
	return t
}

func (p *PubSubPublisher) Publish(ctx context.Context, m eventports.OutboxMessage) error {
	result := p.topic(m.Topic).Publish(ctx, &pubsub.Message{
		Data:        m.Payload,
		OrderingKey: m.Key,
		Attributes: map[string]string{
			"event_type": m.EventType,
			"outbox_id":  strconv.FormatInt(m.OutboxID, 10),
		},
	})
	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("pubsub publish: %w", err)
	}
	return nil
}

func (p *PubSubPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, t := range p.topics {
		t.Stop()
	}
	_ = p.client.Close()
}
