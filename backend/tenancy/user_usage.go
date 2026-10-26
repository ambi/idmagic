package tenancy

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/internal/usecases"
)

// RecountUserUsage は、count が数えた使用量を占める User の件数で User の使用量を置き換える。
// count は使用量を固定している間に呼ばれる。
func (m Module) RecountUserUsage(ctx context.Context, tenantID string, count func(context.Context) (int, error)) error {
	return usecases.RecountUserUsage(ctx, m.QuotaRepo, tenantID, count)
}
