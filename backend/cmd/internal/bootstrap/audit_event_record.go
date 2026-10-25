package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	auditusecases "github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// AuditFailureMetrics は本文や識別子を指標へ渡さず、失敗した段階だけを記録する。
type AuditFailureMetrics interface {
	RecordAuditFailure(stage string)
}

func NewAuditEventRecord(e spec.DomainEvent) (*auditports.AuditEventRecord, error) {
	wire, err := spec.MarshalDomainEvent(e)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(wire, &payload); err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	rec := &auditports.AuditEventRecord{
		ID:         id,
		Type:       e.EventType(),
		OccurredAt: e.OccurredAt(),
		Payload:    payload,
	}
	if tenantID, ok := payload["tenantId"].(string); ok {
		rec.TenantID = tenantID
	}
	if rec.Payload == nil {
		return nil, fmt.Errorf("audit event %s: empty payload", e.EventType())
	}
	rec.SearchAttributes = auditusecases.ExtractSearchAttributes(rec)
	return rec, nil
}

// NewEmitFunc builds the shared "write DomainEvent to EventSink, then mirror it
// into AuditEventRepo" closure (audit.DomainEventsAreAuditedRegardlessOfProcess
// invariant, wi-205). Every process that executes admin/user-facing use cases
// (idmagic, idmagic-worker, ...) must wire its use case Emit dependency through
// this helper, so audit coverage doesn't depend on which process happens to run
// the use case.
func (d *Dependencies) NewEmitFunc(logger logging.Logger) func(spec.DomainEvent) {
	var emit func(spec.DomainEvent)
	emit = func(event spec.DomainEvent) {
		eventCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := d.OAuth2.EventSink.Emit(eventCtx, event); err != nil {
			logger.Error(eventCtx, "event sink emit failed", "error", err)
		}
		// アカウントのセキュリティ通知は同じ配信点から購読する (wi-90)。監査への
		// 書き込みと同じワイヤ表現を見るので、通知の対象は emit 済みの事実に一致する。
		// 送信自体は切り離して走るため、この呼び出しは SMTP を待たない。
		d.dispatchSecurityNotification(logger, emit, event)
		if err := d.Audit.ObserveExportJob(eventCtx, d.Jobs.Repo, event, emit); err != nil {
			logger.Error(eventCtx, "audit export lifecycle observation failed", "error", err)
		}
		if d.Audit.AuditEventRepo == nil {
			return
		}
		rec, err := NewAuditEventRecord(event)
		if err != nil {
			if d.AuditFailures != nil {
				d.AuditFailures.RecordAuditFailure("conversion")
			}
			logger.Error(eventCtx, "audit event conversion failed; record may be missing",
				"error", err, "audit_stage", "conversion", "event_type", event.EventType(), "occurred_at", event.OccurredAt())
			return
		}
		appendCtx := eventCtx
		if rec.TenantID != "" {
			appendCtx = tenantports.WithTenantID(eventCtx, rec.TenantID)
		}
		if err := d.Audit.AuditEventRepo.Append(appendCtx, rec); err != nil {
			if d.AuditFailures != nil {
				d.AuditFailures.RecordAuditFailure("append")
			}
			logger.Error(appendCtx, "audit event append failed; record may be missing",
				"error", err, "audit_stage", "append", "event_type", rec.Type, "occurred_at", rec.OccurredAt,
				"tenant_id", rec.TenantID, "audit_event_id", rec.ID)
		}
	}
	return emit
}
