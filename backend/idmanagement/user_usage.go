package idmanagement

import (
	"context"
	"errors"
)

// UserQuotaUsage は、テナントの User の使用量を占める User を数える。完全削除の使用量の減算を
// 終えていない Tombstone を含める。
func (m Module) UserQuotaUsage(ctx context.Context, tenantID string) (int64, error) {
	if m.UserRepo == nil {
		return 0, errors.New("user repository is required")
	}
	return m.UserRepo.CountHoldingQuota(ctx, tenantID)
}
