package eventlog

import "context"

// Recorder appends event_logs / event_deliveries rows. Implementations are
// bound to a specific DB handle by their constructor (a pool for standalone
// use, or — to satisfy ADR-094's atomicity invariant — a transaction already
// shared with the business mutation's own repository calls). Recorder itself
// stays transaction-agnostic so this port carries no adapter import; the
// transaction-bound command runner that wires Recorder into a shared
// transaction is wi-184 T003.
type Recorder interface {
	// Append inserts rec into event_logs. Duplicate EventID is an error, not
	// an idempotent no-op: a command records its event exactly once.
	Append(ctx context.Context, rec Record) error

	// AppendDelivery inserts a pending event_deliveries row for eventID.
	// Callers only call this for Classification == ClassificationPublicIntegration
	// records; eventID must already exist in event_logs.
	AppendDelivery(ctx context.Context, eventID string) error
}
