package bootstrap

// 認証 / 監査イベントと CSV の成果物の保持期間 sweep と、猶予期間を過ぎた User の完全削除を
// one-shot batch として動かす。周期と再試行は外部 scheduler が所有する。

import (
	"context"
	"errors"
	"fmt"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
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
	if deps.Tenancy.TenantRepo != nil && deps.IdManagement.UserRepo != nil {
		return purgeExpiredUsers(ctx, deps, now)
	}
	return nil
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
	commands := deps.UserLifecycleCommands(deps.NewEmitFunc(logging.Default()), "system")
	var failures []error
	for _, tenant := range tenants {
		tenantCtx := tenantports.WithTenant(ctx, tenant, "", "")
		if err := userusecases.PurgeExpiredSoftDeleted(tenantCtx, commands.Deps, now); err != nil {
			logging.Error(tenantCtx, "user purge failed", "error", err, "tenant_id", tenant.ID)
			failures = append(failures, fmt.Errorf("purge expired users of tenant %s: %w", tenant.ID, err))
		}
	}
	return errors.Join(failures...)
}
