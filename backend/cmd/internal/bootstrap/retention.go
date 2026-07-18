package bootstrap

// 認証 / 監査イベントの保持期間 sweep を one-shot batch として動かす
// (ADR-045, ADR-124)。周期と再試行は外部 scheduler が所有する。

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
	if audit == nil && buckets == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	policy := authusecases.DefaultRetentionPolicy()
	res, err := authusecases.RunRetentionSweep(ctx, audit, buckets, policy, now)
	if err != nil {
		return err
	}
	logging.Info(ctx, "retention sweep completed",
		"deleted_audit_events", res.AuditEvents, "deleted_buckets", res.Buckets)
	return nil
}
