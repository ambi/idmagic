package tenancy

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/internal/usecases"
)

// ReconcileSoftQuota は保存先の測定値を使用量へ反映し、超過の警告を監査へ送る。
func (m Module) ReconcileSoftQuota(ctx context.Context, tenantID string, measured domain.SoftQuotaUsage, now time.Time, emit func(spec.DomainEvent)) error {
	return usecases.ReconcileSoftQuota(ctx, m.QuotaRepo, tenantID, measured, now, emit)
}
