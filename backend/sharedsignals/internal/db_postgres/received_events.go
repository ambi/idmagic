package db_postgres

import (
	"context"
	"encoding/json"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
)

// ReceivedSecurityEventRepository は inbound SET の受理記録 を PostgreSQL に
// 永続化する。replay 検知は UNIQUE (stream_id, set_jti) と ON CONFLICT DO NOTHING で
// 二重挿入を fail-closed に防ぐ (ExistsByJTI と合わせた事前・事後の二重チェック)。
type ReceivedSecurityEventRepository struct{ Pool sharedpg.DB }

func (r *ReceivedSecurityEventRepository) ExistsByJTI(ctx context.Context, tenantID, streamID, setJTI string) (bool, error) {
	return New(r.Pool).ExistsReceivedSecurityEventByJTI(ctx, ExistsReceivedSecurityEventByJTIParams{
		TenantID: tenantID, StreamID: streamID, SetJti: setJTI,
	})
}

func (r *ReceivedSecurityEventRepository) Save(ctx context.Context, e *ssdomain.ReceivedSecurityEvent) error {
	subjectJSON, err := json.Marshal(e.Subject)
	if err != nil {
		return err
	}
	return New(r.Pool).SaveReceivedSecurityEvent(ctx, SaveReceivedSecurityEventParams{
		ID: e.ID, TenantID: e.TenantID, StreamID: e.StreamID, SetJti: e.SetJTI,
		EventType: string(e.EventType), Subject: subjectJSON,
		VerificationResult: string(e.VerificationResult), ReceivedAt: e.ReceivedAt,
		ReflectedAt: timestamptzOrNilPtr(e.ReflectedAt),
	})
}
