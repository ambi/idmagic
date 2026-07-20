package ports

import "context"

// Publisher delivers transport-neutral outbox messages.
type Publisher interface {
	Publish(ctx context.Context, message OutboxMessage) error
	Name() string
	Close()
}

// OutboxMessage represents one transport-neutral outbox record.
type OutboxMessage struct {
	Topic     string
	Key       string
	Payload   []byte
	EventType string
	OutboxID  int64
}
