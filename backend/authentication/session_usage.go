package authentication

import (
	"context"
	"errors"
	"time"
)

// ActiveSessionQuotaUsage は、tenantID の有効なセッション (未失効かつ now で期限内、確認待ちを含む) の
// 件数を返す。Tenancy がセッションの使用量を置き換える再集計に使う (REQ-TENANCY-051)。
func (m Module) ActiveSessionQuotaUsage(ctx context.Context, tenantID string, now time.Time) (int64, error) {
	if m.SessionStore == nil {
		return 0, errors.New("session store is required")
	}
	return m.SessionStore.CountActive(ctx, tenantID, now)
}
