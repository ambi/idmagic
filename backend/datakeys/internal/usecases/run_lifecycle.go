package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/internal/domain"
	"github.com/ambi/idmagic/backend/shared/logging"
)

// RunTenantDataKeyLifecycle は、定期実行が一つのテナントについて行う処理である。
// 周期を過ぎていればローテーションし、過ぎていなければ、ローテーションの後の投入に失敗して
// 残った未移行の行を回収する。DEK を持たないテナントには、初回の暗号化が作るので何もしない。
func RunTenantDataKeyLifecycle(ctx context.Context, deps Deps, tenantID string, now time.Time, cadence time.Duration) error {
	active, err := deps.Repository.FindActive(ctx, tenantID)
	if errors.Is(err, domain.ErrNoActiveDataKey) {
		return nil
	}
	if err != nil {
		return err
	}
	if domain.RotationDue(active, now, cadence) {
		_, err := RotateTenantDataKey(ctx, deps, tenantID, now)
		return err
	}
	if deps.Migrators == nil || deps.Jobs == nil {
		return nil
	}
	for _, name := range deps.Migrators.Names() {
		migrator, ok := deps.Migrators.Lookup(name)
		if !ok {
			continue
		}
		pending, err := migrator.PendingCount(ctx, tenantID, active.Version)
		if err != nil {
			return err
		}
		if pending == 0 {
			continue
		}
		if err := EnqueueReencryptionJob(ctx, deps.Jobs, tenantID, name, now); err != nil {
			// 次回の実行で回収するので、ほかの migrator とテナントの処理を止めない。
			logging.Warn(ctx, "datakeys: failed to enqueue reencryption job from lifecycle", "error", err, "tenant_id", tenantID, "migrator", name)
		}
	}
	return nil
}
