package bootstrap

import (
	"context"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/sharedsignals"
	"github.com/jackc/pgx/v5"
)

func auditCapture(rec *auditports.AuditEventRecord) sharedsignals.AuditCapture {
	return sharedsignals.AuditCapture{EventID: rec.ID, EventType: rec.Type, TenantID: rec.TenantID, OccurredAt: rec.OccurredAt, Payload: rec.Payload}
}

// captureAuditRecordInTx は監査の記録と同じ Tx で SSF の配送を作る。Audit と SharedSignals は
// 互いを import せず、ここで結ぶ。
func captureAuditRecordInTx(ctx context.Context, tx pgx.Tx, rec *auditports.AuditEventRecord) error {
	return sharedsignals.CaptureAuditEventInTx(ctx, tx, auditCapture(rec), time.Now().UTC())
}

// captureAuditRecordInMemory はメモリの保存先の経路で同じ捕捉を行う。
func captureAuditRecordInMemory(module sharedsignals.Module) func(context.Context, *auditports.AuditEventRecord) error {
	return func(ctx context.Context, rec *auditports.AuditEventRecord) error {
		return module.CaptureAuditEvent(ctx, auditCapture(rec), time.Now().UTC())
	}
}
