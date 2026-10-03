package bootstrap

// 認証 / 監査イベントと CSV の成果物の保持期間 sweep を one-shot batch として動かす。
// 周期と再試行は外部 scheduler が所有する。

import (
	"context"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// RunRetentionSweepOnce は保持期間境界を現在時刻で一度だけ適用する。
func RunRetentionSweepOnce(ctx context.Context, deps *Dependencies, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	audit, _ := deps.Audit.AuditEventRepo.(authusecases.AuditEventPurger)
	buckets, _ := deps.Authentication.AuthEventBucketStore.(authusecases.AuthEventBucketPurger)
	sessions, _ := deps.Authentication.SessionStore.(authusecases.SessionPurger)
	knownDevices, _ := deps.Authentication.KnownSignInDeviceRepo.(authusecases.KnownDevicePurger)
	stores := authusecases.RetentionStores{
		Audit: audit, Buckets: buckets, Sessions: sessions, KnownDevices: knownDevices,
	}
	if !stores.Empty() {
		res, err := authusecases.RunRetentionSweep(ctx, stores, authusecases.DefaultRetentionPolicy(), now)
		if err != nil {
			return err
		}
		logging.Info(ctx, "retention sweep completed",
			"deleted_audit_events", res.AuditEvents, "deleted_buckets", res.Buckets,
			"deleted_sessions", res.Sessions, "deleted_known_devices", res.KnownDevices)
	}
	if artifacts, ok := deps.IdManagement.CSVArtifacts.(idmports.CSVArtifactPurger); ok {
		deleted, err := idmusecases.PurgeExpiredCSVArtifacts(ctx, artifacts, now)
		if err != nil {
			return err
		}
		logging.Info(ctx, "csv artifact retention sweep completed", "deleted_csv_artifacts", deleted)
	}
	return nil
}
