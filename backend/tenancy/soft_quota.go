package tenancy

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/internal/usecases"
)

// ReconcileSoftQuota は保存先の測定値を使用量へ反映し、超過の警告を監査へ送る。
func (m Module) ReconcileSoftQuota(ctx context.Context, tenantID string, measured tenancydomain.SoftQuotaUsage, now time.Time, emit func(spec.DomainEvent)) error {
	return tenancyusecases.ReconcileSoftQuota(ctx, m.QuotaRepo, tenantID, measured, now, emit)
}
