package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// QuotaRepository provides atomic usage counters for Tenant resources (wi-160).
type QuotaRepository interface {
	// CheckAndIncrement atomically increments the usage counter for the given resource.
	// Hard の上限を超える加算だけを QuotaExceededError で拒否し、Soft の超過では拒否しない。
	CheckAndIncrement(ctx context.Context, tenantID, resource string, delta int) error

	// Decrement atomically decreases the usage counter for the given resource.
	// Usage must not go below 0.
	Decrement(ctx context.Context, tenantID, resource string, delta int) error

	// SetQuota explicitly sets the quota for a tenant.
	SetQuota(ctx context.Context, tenantID string, quota *domain.TenantQuota) error

	// GetQuota retrieves the quota for a tenant.
	GetQuota(ctx context.Context, tenantID string) (*domain.TenantQuota, error)

	// GetUsage retrieves the current usage for a tenant.
	GetUsage(ctx context.Context, tenantID string) (*domain.TenantUsage, error)

	// SetSoftUsage は測定した二列だけを同時に置換し、Hard の使用量を維持する。
	SetSoftUsage(ctx context.Context, tenantID string, usage domain.SoftQuotaUsage) error

	// RecountUsage は resource の使用量を固定したまま count を呼び、返った件数で使用量を置き換え、
	// 置き換える前の使用量を返す。固定している間の加算と減算は、置き換えの後に適用される。
	// count は固定の間に呼ばれるので、QuotaRepository を呼ばない。いまは ResourceUsers だけを受け付ける。
	RecountUsage(ctx context.Context, tenantID, resource string, count func(context.Context) (int, error)) (previous int, err error)
}
