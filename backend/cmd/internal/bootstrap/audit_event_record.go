package bootstrap

// newAuditEventRecord は DomainEvent を AuditEventRepository に保存可能な
// 形 (SCL `AdminAuditEventResponse` の双子) に変換する。tenant_id は
// payload に tenantId が存在する場合のみ抽出し、無い場合は空文字を残す
// (admin の所属テナント絞り込みでは引っかからない)。
//
// SCL events セクションで TenantID を持つのは Tenant ライフサイクル系のみ。
// それ以外の event は将来 TenantID を載せれば、本変換だけで audit 経路から
// 見えるようになる。

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	auditusecases "github.com/ambi/idmagic/backend/audit/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

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
	return func(event spec.DomainEvent) {
		eventCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := d.OAuth2.EventSink.Emit(eventCtx, event); err != nil {
			logger.Error(eventCtx, "event sink emit failed", "error", err)
		}
		if d.Audit.AuditEventRepo == nil {
			return
		}
		rec, err := NewAuditEventRecord(event)
		if err != nil {
			logger.Error(eventCtx, "audit event conversion failed; reconciliation required", "error", err, "event_type", event.EventType())
			return
		}
		appendCtx := eventCtx
		if rec.TenantID != "" {
			appendCtx = tenancy.WithTenant(eventCtx, &tenancydomain.Tenant{ID: rec.TenantID}, "", "")
		}
		if err := d.Audit.AuditEventRepo.Append(appendCtx, rec); err != nil {
			logger.Error(appendCtx, "audit event append failed; reconciliation required", "error", err, "event_type", event.EventType(), "tenant_id", rec.TenantID)
		}
	}
}
