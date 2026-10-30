package db_postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// CaptureAuditEventInTx は監査の記録と同じ Tx で配送を作る。テナントが UUID でない記録は
// ストリームを持たないので何もしない。
func CaptureAuditEventInTx(ctx context.Context, tx pgx.Tx, event ssdomain.AuditSetEvent, now time.Time) error {
	if !ssdomain.CapturableAuditType(event.EventType) {
		return nil
	}
	var tenant pgtype.UUID
	if err := tenant.Scan(event.TenantID); err != nil {
		return nil //nolint:nilerr // 制御面の記録など、テナントを持たない記録は捕捉の対象でない。
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return New(tx).CaptureAuditEventDeliveries(ctx, CaptureAuditEventDeliveriesParams{
		TenantID: event.TenantID, EventType: event.EventType, AuditEvent: raw, Now: now,
	})
}

func (r *SecurityEventDeliveryRepository) ClaimDue(ctx context.Context, now, leaseUntil time.Time, perStream, limit int) ([]*ssdomain.SecurityEventDelivery, error) {
	rows, err := New(r.Pool).ClaimDueSecurityEventDeliveries(ctx, ClaimDueSecurityEventDeliveriesParams{
		Now: now, LeaseUntil: leaseUntil, PerStream: int64(perStream),
		RowLimit: int32(limit), //nolint:gosec // G115: caller-supplied small batch size
	})
	if err != nil {
		return nil, err
	}
	out := make([]*ssdomain.SecurityEventDelivery, 0, len(rows))
	for _, row := range rows {
		d, err := deliveryFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *SecurityEventDeliveryRepository) PurgeSettled(ctx context.Context, before time.Time) (int64, error) {
	return New(r.Pool).PurgeSettledSecurityEventDeliveries(ctx, before)
}

func (r *SsfStreamRepository) RecordDeliveryFailure(ctx context.Context, tenantID, id string, now time.Time) (bool, error) {
	row, err := New(r.Pool).RecordSsfStreamDeliveryFailure(ctx, RecordSsfStreamDeliveryFailureParams{
		TenantID: tenantID, ID: id, Now: now, Threshold: ssdomain.AutoDisableFailures,
		MinElapsedSeconds: ssdomain.AutoDisableElapsed.Seconds(),
	})
	if err != nil {
		if err == pgx.ErrNoRows { //nolint:errorlint // pgx は包まずに返す。
			return false, nil
		}
		return false, err
	}
	return row.PreviousStatus == string(ssdomain.SsfStreamStatusEnabled) && row.NewStatus == string(ssdomain.SsfStreamStatusDisabled), nil
}

func (r *SsfStreamRepository) RecordDeliverySuccess(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).ClearSsfStreamDeliveryFailures(ctx, ClearSsfStreamDeliveryFailuresParams{TenantID: tenantID, ID: id})
}
