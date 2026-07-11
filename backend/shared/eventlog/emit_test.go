package eventlog_test

// NewBridgingEmit is the temporary bridge described in emit.go: until wi-185's
// audit projection and wi-190's relay read from event_logs/event_deliveries
// directly, a migrated mutation must also keep feeding the legacy
// audit_events/outbox path or it silently vanishes from the admin audit UI
// (and, for public_integration events, from Kafka).

import (
	"context"
	"testing"
	"time"

	memoryeventlog "github.com/ambi/idmagic/backend/shared/adapters/persistence/memory/eventlog"
	"github.com/ambi/idmagic/backend/shared/eventlog"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestNewBridgingEmitAppendsToEventLogAndCallsLegacy(t *testing.T) {
	recorder := memoryeventlog.New()
	var legacyCalls []spec.DomainEvent
	legacy := func(e spec.DomainEvent) { legacyCalls = append(legacyCalls, e) }

	emit := eventlog.NewBridgingEmit(context.Background(), recorder, "corr-1", legacy)
	event := &spec.PasswordChanged{At: time.Now().UTC(), TenantID: "tenant-1", UserID: "user-1"}
	if err := emit(event); err != nil {
		t.Fatalf("emit: %v", err)
	}

	if len(legacyCalls) != 1 || legacyCalls[0] != spec.DomainEvent(event) {
		t.Fatalf("legacy callback not invoked with the same event: %#v", legacyCalls)
	}
	if got := recorder.Count(); got != 1 {
		t.Fatalf("event_logs row count = %d, want 1 (the transactional append)", got)
	}
}

func TestNewBridgingEmitSkipsLegacyWhenEventLogAppendFails(t *testing.T) {
	recorder := memoryeventlog.New()
	var legacyCalled bool
	legacy := func(spec.DomainEvent) { legacyCalled = true }

	emit := eventlog.NewBridgingEmit(context.Background(), recorder, "corr-1", legacy)
	if err := emit(unclassifiedEvent{}); err == nil {
		t.Fatal("expected ToRecord to reject an unclassified DomainEvent")
	}
	if legacyCalled {
		t.Fatal("legacy must not run when the transactional event_logs append fails")
	}
	if got := recorder.Count(); got != 0 {
		t.Fatalf("event_logs row count = %d, want 0 (append must not have happened)", got)
	}
}

type unclassifiedEvent struct{}

func (unclassifiedEvent) EventType() string     { return "NotARealDomainEventType" }
func (unclassifiedEvent) OccurredAt() time.Time { return time.Now().UTC() }
