package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/ports"
)

func ReconcileSoftQuota(ctx context.Context, repository ports.QuotaRepository, tenantID string, measured domain.SoftQuotaUsage, now time.Time, emit func(spec.DomainEvent)) error {
	if repository == nil || emit == nil {
		return errors.New("soft quota: repository and event emitter are required")
	}
	if err := measured.Validate(); err != nil {
		return err
	}
	quota, err := repository.GetQuota(ctx, tenantID)
	if err != nil {
		return err
	}
	if err := repository.SetSoftUsage(ctx, tenantID, measured); err != nil {
		return err
	}
	for _, resource := range measured.ExceededResources(quota) {
		usage := measured.AuditEventsRetained
		if resource == domain.ResourceExportArtifactsBytes {
			usage = measured.ExportArtifactsBytes
		}
		logging.Warn(ctx, "soft quota exceeded", "tenant_id", tenantID, "resource", resource, "usage", usage, "limit", quota.EffectiveLimit(resource))
		emit(&domain.QuotaExceeded{At: now, TenantID: tenantID, Resource: resource, HardLimit: false})
	}
	return nil
}
