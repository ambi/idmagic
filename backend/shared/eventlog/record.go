// Package eventlog holds the System context's cross-cutting event log domain
// types (ADR-094, wi-184): the immutable event_logs record and its Kafka
// delivery state. Every bounded context's transaction-bound command writes
// through this shape. It lives in backend/shared rather than any single
// context because recording an event_logs row is a technical common part
// (ADR-090) used by every context's write path, not a domain concept the
// read-only Audit context (wi-146) owns.
package eventlog

import "time"

// Classification is the SCL DomainEventClassification enum
// (spec/contexts/system.yaml).
type Classification string

const (
	ClassificationPublicIntegration Classification = "public_integration"
	ClassificationAuditOnly         Classification = "audit_only"
	ClassificationTelemetry         Classification = "telemetry"
)

// DeliveryStatus is the SCL EventDeliveryStatus enum
// (spec/contexts/system.yaml).
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
)

// Record is a single event_logs row (SCL EventLogRecord). Rows are inserted
// once, in the same transaction as the business mutation they record, and
// are never updated afterward.
type Record struct {
	EventID        string
	TenantID       string
	Type           string
	Classification Classification
	Actor          string // optional; "" means absent
	Subject        string // optional; "" means absent
	CorrelationID  string
	OccurredAt     time.Time
	Payload        map[string]any
}

// Delivery is a single event_deliveries row (SCL EventDeliveryRecord).
type Delivery struct {
	EventID     string
	Status      DeliveryStatus
	Attempts    int
	LastError   string
	DeliveredAt *time.Time
	UpdatedAt   time.Time
}
