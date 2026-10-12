package db_postgres

import (
	"context"
	"encoding/json"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// SecurityEventDeliveryRepository は SecurityEventDelivery (outbound SET outbox)
// を PostgreSQL に永続化する。set_payload には SecurityEventToken 全体を
// JSON で埋め込む (別テーブルへ正規化しない: outbox 行は当該 SET の配送状態そのもの)。
type SecurityEventDeliveryRepository struct{ Pool sharedpg.DB }

func deliveryFromRow(row *SecurityEventDelivery) (*ssdomain.SecurityEventDelivery, error) {
	var set ssdomain.SecurityEventToken
	if err := json.Unmarshal(row.SetPayload, &set); err != nil {
		return nil, err
	}
	return &ssdomain.SecurityEventDelivery{
		ID: row.ID, TenantID: row.TenantID, StreamID: row.StreamID, SetJTI: row.SetJti,
		Set: set, Status: ssdomain.SecurityEventDeliveryStatus(row.Status),
		AttemptCount: int(row.AttemptCount), NextAttemptAt: timestamptzPtrOrNil(row.NextAttemptAt),
		LastError: textPtrOrNil(row.LastError), CreatedAt: row.CreatedAt,
		DeliveredAt: timestamptzPtrOrNil(row.DeliveredAt),
	}, nil
}

func (r *SecurityEventDeliveryRepository) ListByStream(ctx context.Context, tenantID, streamID string) ([]*ssdomain.SecurityEventDelivery, error) {
	rows, err := New(r.Pool).ListSecurityEventDeliveriesByStream(ctx, ListSecurityEventDeliveriesByStreamParams{TenantID: tenantID, StreamID: streamID})
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

func (r *SecurityEventDeliveryRepository) ListDue(ctx context.Context, now time.Time, limit int) ([]*ssdomain.SecurityEventDelivery, error) {
	rows, err := New(r.Pool).ListDueSecurityEventDeliveries(ctx, ListDueSecurityEventDeliveriesParams{
		NextAttemptAt: timestamptzOrNilPtr(&now), RowLimit: int32(limit), //nolint:gosec // G115: caller-supplied small batch size
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

func (r *SecurityEventDeliveryRepository) Save(ctx context.Context, d *ssdomain.SecurityEventDelivery) error {
	payload, err := json.Marshal(d.Set)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveSecurityEventDelivery(ctx, SaveSecurityEventDeliveryParams{
		ID: d.ID, TenantID: d.TenantID, StreamID: d.StreamID, SetJti: d.SetJTI, SetPayload: payload,
		Status: string(d.Status), AttemptCount: int32(d.AttemptCount), //nolint:gosec // G115: bounded by max_delivery_attempts
		NextAttemptAt: timestamptzOrNilPtr(d.NextAttemptAt), LastError: textOrNilPtr(d.LastError),
		CreatedAt: d.CreatedAt, DeliveredAt: timestamptzOrNilPtr(d.DeliveredAt),
	})
}
