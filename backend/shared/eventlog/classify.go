package eventlog

import (
	"encoding/json"
	"fmt"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// classification is the ADR-094 DomainEvent -> Classification catalog.
// wi-184 T003 seeds it with the events emitted by the mutations already
// migrated to the transaction runner (identitymanagement admin user
// create/update/disable/enable, authentication ChangePassword); wi-184 T004
// makes this exhaustive and CI-checked so every DomainEvent has a routing
// decision. Values mirror the current outbox eventTopics map
// (backend/oauth2/adapters/persistence/postgres/outbox.go): only
// PasswordChanged is routed to Kafka today, so it is the only
// public_integration entry here — the rest currently only reach the audit
// trail.
var classification = map[string]Classification{
	"UserCreated":               ClassificationAuditOnly,
	"UserUpdated":               ClassificationAuditOnly,
	"UserDisabled":              ClassificationAuditOnly,
	"UserEnabled":               ClassificationAuditOnly,
	"UserRequiredActionCleared": ClassificationAuditOnly,
	"PasswordChanged":           ClassificationPublicIntegration,
}

// ToRecord converts event into a Record ready for Recorder.Append. eventID
// is the caller-supplied idempotency key (a fresh UUIDv4 per event);
// correlationID ties every event recorded within one HTTP request/command
// together.
//
// tenantId / actorUserId / targetUserId (or userId) are read generically
// from the event's own JSON wire form (spec.MarshalDomainEvent) using the
// same convention backend/bootstrap/audit_event_record.go already relies on
// for tenantId: every DomainEvent that carries an actor/subject tags them
// with these JSON keys, so this does not need a type switch per event.
//
// Classification is the one thing that cannot be inferred generically: it
// is an explicit routing decision, not a structural fact of the payload. An
// event type missing from the classification map fails closed with an
// error instead of guessing, so an unmigrated command is a hard failure to
// notice (and roll back) rather than a silently misrouted event.
func ToRecord(event spec.DomainEvent, eventID, correlationID string) (Record, error) {
	wire, err := spec.MarshalDomainEvent(event)
	if err != nil {
		return Record{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(wire, &payload); err != nil {
		return Record{}, err
	}
	class, ok := classification[event.EventType()]
	if !ok {
		return Record{}, fmt.Errorf(
			"eventlog: %s is not classified (wi-184 T004 tracks the full DomainEvent catalog)", event.EventType(),
		)
	}
	rec := Record{
		EventID:        eventID,
		Type:           event.EventType(),
		Classification: class,
		CorrelationID:  correlationID,
		OccurredAt:     event.OccurredAt(),
		Payload:        payload,
	}
	if tenantID, ok := payload["tenantId"].(string); ok {
		rec.TenantID = tenantID
	}
	if actor, ok := payload["actorUserId"].(string); ok {
		rec.Actor = actor
	}
	if target, ok := payload["targetUserId"].(string); ok {
		rec.Subject = target
	} else if userID, ok := payload["userId"].(string); ok {
		rec.Subject = userID
	}
	return rec, nil
}
