package tenancy

import (
	"context"

	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/internal/usecases"
)

// RecountUserUsage は、count が数えた使用量を占める User の件数で User の使用量を置き換える。
// count は使用量を固定している間に呼ばれる。
func (m Module) RecountUserUsage(ctx context.Context, tenantID string, count func(context.Context) (int, error)) error {
	return tenancyusecases.RecountUserUsage(ctx, m.QuotaRepo, tenantID, count)
}

// RecountActiveSessionUsage は、count が数えた有効なセッションの件数でセッションの使用量を
// 置き換える。count は使用量を固定している間に呼ばれる。
func (m Module) RecountActiveSessionUsage(ctx context.Context, tenantID string, count func(context.Context) (int, error)) error {
	return tenancyusecases.RecountActiveSessionUsage(ctx, m.QuotaRepo, tenantID, count)
}
