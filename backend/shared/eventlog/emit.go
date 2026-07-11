package eventlog

import (
	"context"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// NewEmit returns a function suitable for a usecase Deps.Emit field
// (func(spec.DomainEvent) error) that appends the event to event_logs
// within ctx's active transaction and, for classification ==
// public_integration, also inserts a pending event_deliveries row so relay
// (wi-184 T005) can pick it up once it exists. correlationID ties every
// event emitted through the returned function together (typically the HTTP
// request id).
//
// Callers construct this once per command inside the transaction runner's
// fn (see backend/shared/adapters/persistence/postgres.Runner.Run), passing
// the tx-carrying ctx so recorder observes the same transaction as the
// business mutation's own repository calls (ADR-094
// EventLogAtomicWithBusinessState).
func NewEmit(ctx context.Context, recorder Recorder, correlationID string) func(spec.DomainEvent) error {
	return func(event spec.DomainEvent) error {
		eventID, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		rec, err := ToRecord(event, eventID, correlationID)
		if err != nil {
			return err
		}
		if err := recorder.Append(ctx, rec); err != nil {
			return err
		}
		if rec.Classification == ClassificationPublicIntegration {
			if err := recorder.AppendDelivery(ctx, rec.EventID); err != nil {
				return err
			}
		}
		return nil
	}
}

// NewBridgingEmit wraps NewEmit so it also invokes legacy — the pre-existing
// fire-and-forget emit (support.Deps.Emit) that feeds audit_events and, for
// Kafka-routed types, outbox — after a successful transactional append.
//
// Nothing reads event_logs/event_deliveries yet: the admin audit UI queries
// audit_events, and relay still only drains outbox. Without this bridge,
// migrating a mutation's Emit to NewEmit alone would silently stop it from
// appearing in the audit log and (for public_integration events) from
// reaching Kafka, even though ADR-094's atomicity guarantee is met. legacy
// is called outside the transaction and its outcome does not affect it
// (fire-and-forget, matching its pre-existing behavior) — only the
// event_logs append can roll back the command.
//
// This is a deliberate temporary duplication, not the target architecture:
// remove it once wi-185's audit projection reads from event_logs directly
// and wi-190's relay reads from event_deliveries, and switch callers back to
// NewEmit.
func NewBridgingEmit(ctx context.Context, recorder Recorder, correlationID string, legacy func(spec.DomainEvent)) func(spec.DomainEvent) error {
	txEmit := NewEmit(ctx, recorder, correlationID)
	return func(event spec.DomainEvent) error {
		if err := txEmit(event); err != nil {
			return err
		}
		if legacy != nil {
			legacy(event)
		}
		return nil
	}
}
