package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// recountActiveSessionUsages は、各テナントのセッションの使用量を、Authentication が now で数えた
// 有効なセッションの件数で置き換える。期限切れ、完全削除、受信した SET で終わったセッションは
// 減算を持たないので、ここで使用量から外す (REQ-TENANCY-051)。一つのテナントの失敗で残りの
// テナントを止めない。
func recountActiveSessionUsages(ctx context.Context, deps *Dependencies, now time.Time) error {
	// Quota またはセッションのない部分的なテスト構成は、再集計の対象を持たない。
	if deps.Tenancy.QuotaRepo == nil || deps.Tenancy.TenantRepo == nil || deps.Authentication.SessionStore == nil {
		return nil
	}
	tenants, err := deps.Tenancy.TenantRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	var failures []error
	for _, tenant := range tenants {
		tenantCtx := tenantports.WithTenant(ctx, tenant, "", "")
		count := func(ctx context.Context) (int, error) {
			n, err := deps.Authentication.ActiveSessionQuotaUsage(ctx, tenant.ID, now)
			if err != nil {
				return 0, err
			}
			if n > math.MaxInt {
				return 0, errors.New("active session usage recount: count overflows int")
			}
			return int(n), nil
		}
		if err := deps.Tenancy.RecountActiveSessionUsage(tenantCtx, tenant.ID, count); err != nil {
			logging.Error(tenantCtx, "active session usage recount failed", "tenant_id", tenant.ID, "error", err)
			failures = append(failures, fmt.Errorf("active session usage of tenant %s: %w", tenant.ID, err))
		}
	}
	return errors.Join(failures...)
}
