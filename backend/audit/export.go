package audit

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/internal/usecases"
	"github.com/ambi/idmagic/backend/audit/ports"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const KindAuditExport = usecases.KindAuditExport

type (
	ExportDeps  = usecases.ExportDeps
	ExportScope = usecases.ExportScope
	ExportView  = usecases.ExportView
)

func StartExport(ctx context.Context, d ExportDeps, p ports.ExportParams) (*ExportView, error) {
	return usecases.StartExport(ctx, d, p)
}

func GetExport(ctx context.Context, d ExportDeps, s ExportScope, id string) (*ExportView, error) {
	return usecases.GetExport(ctx, d, s, id)
}

func DownloadExport(ctx context.Context, d ExportDeps, s ExportScope, id string) (io.ReadCloser, ports.ExportArtifact, error) {
	return usecases.DownloadExport(ctx, d, s, id)
}

type ExportJobDeps struct {
	Jobs jobsports.JobRepository
	Emit func(spec.DomainEvent)
}

func (m Module) RegisterExportHandler(registry jobsports.HandlerRegistrar, d ExportJobDeps) {
	registry.Register(KindAuditExport, usecases.ExportHandler(ExportDeps{Events: m.AuditEventRepo, Artifacts: m.ExportArtifacts, Jobs: d.Jobs, Codec: m.ExportCodec, Emit: d.Emit, Metrics: m.metrics}))
}

func (m Module) ObserveExportJob(ctx context.Context, jobs jobsports.JobRepository, event spec.DomainEvent, emit func(spec.DomainEvent)) error {
	return usecases.ObserveExportJob(ctx, ExportDeps{Jobs: jobs, Emit: emit, Metrics: m.metrics}, event)
}

func (m Module) ExportUsage(ctx context.Context, tenantID string) (int64, error) {
	if m.ExportArtifacts == nil {
		return 0, nil
	}
	return m.ExportArtifacts.Usage(ctx, tenantID)
}

func (m Module) PurgeExpiredExports(ctx context.Context, jobs jobsports.JobRepository, now time.Time, emit func(spec.DomainEvent)) error {
	if m.ExportArtifacts == nil {
		return nil
	}
	for {
		expired, err := m.ExportArtifacts.Expire(ctx, now)
		if err != nil {
			return err
		}
		for _, artifact := range expired {
			base := domain.ExportEvent{At: now, TenantID: artifact.TenantID, ExportID: artifact.JobID, Scope: "tenant"}
			if jobs != nil {
				if job, err := jobs.Get(ctx, artifact.JobID); err == nil {
					var params ports.ExportParams
					if json.Unmarshal(job.Params, &params) == nil {
						base.ActorUserID = params.RequestedBy
						if params.Query.AllTenants {
							base.Scope = "system"
						}
					}
				}
			}
			emit(&domain.AuditExportExpired{ExportEvent: base})
			m.metrics.Record("expired", "expired")
		}
		if len(expired) < 1000 {
			return nil
		}
	}
}
