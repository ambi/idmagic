package bootstrap

// 認証 / 監査イベントの保持期間 sweep を one-shot batch として動かす。
// 周期と再試行は外部 scheduler が所有する。

import (
	"context"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// RunRetentionSweepOnce は保持期間境界を現在時刻で一度だけ適用する。
func RunRetentionSweepOnce(ctx context.Context, deps *Dependencies, now time.Time) error {
	audit, _ := deps.Audit.AuditEventRepo.(authusecases.AuditEventPurger)
	buckets, _ := deps.Authentication.AuthEventBucketStore.(authusecases.AuthEventBucketPurger)
	sessions, _ := deps.Authentication.SessionStore.(authusecases.SessionPurger)
	knownDevices, _ := deps.Authentication.KnownSignInDeviceRepo.(authusecases.KnownDevicePurger)
	stores := authusecases.RetentionStores{
		Audit: audit, Buckets: buckets, Sessions: sessions, KnownDevices: knownDevices,
	}
	if stores.Empty() {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	policy := authusecases.DefaultRetentionPolicy()
	res, err := authusecases.RunRetentionSweep(ctx, stores, policy, now)
	if err != nil {
		return err
	}
	logging.Info(ctx, "retention sweep completed",
		"deleted_audit_events", res.AuditEvents, "deleted_buckets", res.Buckets,
		"deleted_sessions", res.Sessions, "deleted_known_devices", res.KnownDevices)
	return nil
}
