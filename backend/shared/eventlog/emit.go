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
