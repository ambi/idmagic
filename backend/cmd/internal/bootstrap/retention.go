package bootstrap

// 認証 / 監査イベントと CSV の成果物の保持期間 sweep、猶予期間を過ぎた User の完全削除、
// User の使用量の再集計と Soft Quota の照合を one-shot batch として動かす。周期と再試行は外部 scheduler が所有する。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// RunRetentionSweepOnce は保持期間境界を現在時刻で一度だけ適用する。
func RunRetentionSweepOnce(ctx context.Context, deps *Dependencies, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	res, ran, err := deps.Authentication.RunRetentionSweep(ctx, deps.Audit.AuditEventRepo, now)
	if err != nil {
		return err
	}
	if ran {
		logging.Info(ctx, "retention sweep completed",
			"deleted_audit_events", res.AuditEvents, "deleted_buckets", res.Buckets,
			"deleted_sessions", res.Sessions, "deleted_known_devices", res.KnownDevices)
	}
	deleted, purged, err := deps.IdManagement.PurgeExpiredCSVArtifacts(ctx, now)
	if err != nil {
		return err
	}
	if purged {
		logging.Info(ctx, "csv artifact retention sweep completed", "deleted_csv_artifacts", deleted)
	}
	//nolint:contextcheck // バッチの監査は独立した期限付きの配信点を使う。
	if err := deps.Audit.PurgeExpiredExports(ctx, deps.Jobs.Repo, now, deps.NewEmitFunc(logging.Default())); err != nil {
		return err
	}
	if deps.SharedSignals.DeliveryRepo != nil {
		deletedDeliveries, err := deps.SharedSignals.PurgeExpiredDeliveries(ctx, now)
		if err != nil {
			return err
		}
		logging.Info(ctx, "security event delivery retention sweep completed", "deleted_security_event_deliveries", deletedDeliveries)
	}
	var purgeErr error
	if deps.Tenancy.TenantRepo != nil && deps.IdManagement.UserRepo != nil {
		purgeErr = purgeExpiredUsers(ctx, deps, now)
	}
	// 完全削除の後に数え、この実行で終えた完全削除の減算を反映した使用量にする。
	return errors.Join(purgeErr, recountUserUsages(ctx, deps), reconcileSoftQuotas(ctx, deps, now))
}

// purgeExpiredUsers は、各テナントで猶予期間を過ぎた削除予約の User を完全削除し、匿名化の後に
// 失敗した完全削除を再開する。依存は管理 API の外から User を止める経路と同じ集合にする。
// 一つのテナントの失敗で残りのテナントを止めない。
func purgeExpiredUsers(ctx context.Context, deps *Dependencies, now time.Time) error {
	tenants, err := deps.Tenancy.TenantRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	//nolint:contextcheck // Batch events use the bounded independent audit context.
	commands := deps.IdManagement.UserLifecycleCommands(deps.UserLifecycleDeps(), deps.ReactiveEmit(deps.NewEmitFunc(logging.Default())), "system")
	var failures []error
	for _, tenant := range tenants {
		tenantCtx := tenantports.WithTenant(ctx, tenant, "", "")
		if err := commands.PurgeExpiredSoftDeleted(tenantCtx, now); err != nil {
			logging.Error(tenantCtx, "user purge failed", "error", err, "tenant_id", tenant.ID)
			failures = append(failures, fmt.Errorf("purge expired users of tenant %s: %w", tenant.ID, err))
		}
	}
	return errors.Join(failures...)
}
