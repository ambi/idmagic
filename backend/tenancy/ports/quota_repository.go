package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/tenancy/domain"
)

// QuotaRepository provides atomic usage counters for Tenant resources (wi-160).
type QuotaRepository interface {
	// CheckAndIncrement atomically increments the usage counter for the given resource.
	// It returns domain.QuotaExceededError if the increment would exceed the tenant's quota.
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
}
