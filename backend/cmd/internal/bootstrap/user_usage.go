package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ambi/idmagic/backend/shared/logging"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// recountUserUsages は、各テナントの User の使用量を、IdManagement が数えた使用量を占める User の
// 件数で置き換える。作成と完全削除の途中の失敗で残ったずれを回収する。一つのテナントの失敗で
// 残りのテナントを止めない。
func recountUserUsages(ctx context.Context, deps *Dependencies) error {
	// Quota または User のない部分的なテスト構成は、再集計の対象を持たない。
	if deps.Tenancy.QuotaRepo == nil || deps.Tenancy.TenantRepo == nil || deps.IdManagement.UserRepo == nil {
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
			n, err := deps.IdManagement.UserQuotaUsage(ctx, tenant.ID)
			if err != nil {
				return 0, err
			}
			if n > math.MaxInt {
				return 0, errors.New("user usage recount: count overflows int")
			}
			return int(n), nil
		}
		if err := deps.Tenancy.RecountUserUsage(tenantCtx, tenant.ID, count); err != nil {
			logging.Error(tenantCtx, "user usage recount failed", "tenant_id", tenant.ID, "error", err)
			failures = append(failures, fmt.Errorf("user usage of tenant %s: %w", tenant.ID, err))
		}
	}
	return errors.Join(failures...)
}
