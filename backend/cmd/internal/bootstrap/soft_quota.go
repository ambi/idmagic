package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func reconcileSoftQuotas(ctx context.Context, deps *Dependencies, now time.Time) error {
	// Quota のない部分的なテスト構成は、照合の対象を持たない。
	if deps.Tenancy.QuotaRepo == nil {
		return nil
	}
	if deps.Tenancy.TenantRepo == nil || deps.Audit.AuditEventRepo == nil {
		return errors.New("soft quota: tenant and audit repositories are required")
	}
	tenants, err := deps.Tenancy.TenantRepo.FindAll(ctx)
	if err != nil {
		return err
	}
	//nolint:contextcheck // バッチの監査は独立した期限付きの配信点を使う。
	emit := deps.NewEmitFunc(logging.Default())
	var failures []error
	for _, tenant := range tenants {
		tenantCtx := tenantports.WithTenant(ctx, tenant, "", "")
		measured, err := measureSoftUsage(tenantCtx, deps, tenant.ID)
		if err == nil {
			err = deps.Tenancy.ReconcileSoftQuota(tenantCtx, tenant.ID, measured, now, emit)
		}
		if err != nil {
			logging.Error(tenantCtx, "soft quota reconciliation failed", "tenant_id", tenant.ID, "error", err)
			failures = append(failures, fmt.Errorf("soft quota of tenant %s: %w", tenant.ID, err))
		}
	}
	return errors.Join(failures...)
}

func measureSoftUsage(ctx context.Context, deps *Dependencies, tenantID string) (domain.SoftQuotaUsage, error) {
	count, err := deps.Audit.AuditEventRepo.Count(ctx, auditports.AuditEventQuery{TenantID: tenantID})
	if err != nil {
		return domain.SoftQuotaUsage{}, err
	}
	bytes, err := deps.IdManagement.CSVArtifactUsage(ctx, tenantID)
	if err != nil {
		return domain.SoftQuotaUsage{}, err
	}
	if count > int64(math.MaxInt) || bytes > int64(math.MaxInt) {
		return domain.SoftQuotaUsage{}, errors.New("soft quota: measured usage overflows int")
	}
	return domain.SoftQuotaUsage{AuditEventsRetained: int(count), ExportArtifactsBytes: int(bytes)}, nil
}
