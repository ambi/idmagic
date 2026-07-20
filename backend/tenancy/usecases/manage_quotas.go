package usecases

import (
	"context"

	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// CheckQuotaAndIncrement checks if the tenant has enough quota for the given resource and increments its usage.
// It returns domain.QuotaExceededError if it fails.
func CheckQuotaAndIncrement(ctx context.Context, repo tenantports.QuotaRepository, tenantID, resource string, delta int) error {
	return repo.CheckAndIncrement(ctx, tenantID, resource, delta)
}

// DecrementQuota decreases the usage counter for the given resource.
func DecrementQuota(ctx context.Context, repo tenantports.QuotaRepository, tenantID, resource string, delta int) error {
	return repo.Decrement(ctx, tenantID, resource, delta)
}
