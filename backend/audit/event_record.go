package audit

import (
	"encoding/json"
	"fmt"

	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// NewEventRecord は DomainEvent を、監査へ追記する記録へ変換する。本文は送信表現 (wire) と
// 同じ JSON で、テナントは本文の tenantId から、検索属性は抽出器から決める。
//
// 製品のすべてのプロセスの Emit (cmd/internal/bootstrap) と、同じ組み立てを再現する
// テストの基盤が、同じ変換を通るように Audit が持つ。
func NewEventRecord(e spec.DomainEvent) (*ports.AuditEventRecord, error) {
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
	rec := &ports.AuditEventRecord{
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
	rec.SearchAttributes = usecases.ExtractSearchAttributes(rec)
	return rec, nil
}
