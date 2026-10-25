package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditdomain "github.com/ambi/idmagic/backend/audit/domain"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-AUDIT-011: Jobs の確定済みイベントを正式な配信点が Audit のライフサイクルへ写す。
//spec:covers REQ-TENANCY-048: CSV と監査成果物の合計容量を所有テナントへ照合し、期限削除も反映する。
func TestAuditExportLifecycleAndQuotaWiring(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	sink := &recordingEventSink{}
	deps := &Dependencies{Audit: audit.NewMemoryModule(100), Jobs: jobs.NewMemoryModule(), Tenancy: tenancy.NewMemoryModule(), IdManagement: idmanagement.NewMemoryModule(), OAuth2: oauth2.Module{EventSink: sink}}
	tenantID := tenantdomain.DefaultTenantID
	if err := deps.Tenancy.TenantRepo.Save(ctx, &tenantdomain.Tenant{ID: tenantID, Realm: tenantdomain.DefaultRealm, Status: tenantdomain.TenantStatusActive}); err != nil {
		t.Fatal(err)
	}
	emit := deps.NewEmitFunc(logging.New(io.Discard, slog.LevelInfo, "audit-test", "test"))
	deps.Jobs.Enqueuer = jobs.NewEnqueuer(deps.Jobs.Repo, deps.Tenancy.QuotaRepo, emit)
	view, err := audit.StartExport(ctx, audit.ExportDeps{Artifacts: deps.Audit.ExportArtifacts, Jobs: deps.Jobs.Repo, Enqueuer: deps.Jobs.Enqueuer, Emit: emit}, auditports.ExportParams{Query: auditports.AuditEventQuery{TenantID: tenantID, AllTenants: true}, RequestedBy: "owner", Format: "json", Policy: auditdomain.DefaultExportPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := deps.Jobs.Repo.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC())
	if err != nil || len(claimed) != 1 {
		t.Fatalf("claims=%v err=%v", claimed, err)
	}
	emit(&jobsdomain.JobStarted{At: now, JobID: view.ID, TenantID: tenantID})
	registry := jobs.NewHandlerRegistry()
	deps.Audit.RegisterExportHandler(registry, audit.ExportJobDeps{Jobs: deps.Jobs.Repo, Emit: emit})
	handler, err := registry.Lookup(audit.KindAuditExport)
	if err != nil {
		t.Fatal(err)
	}
	result, err := handler(ctx, claimed[0])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := deps.Jobs.Repo.Complete(ctx, view.ID, "worker", result, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	emit(&jobsdomain.JobSucceeded{At: now, JobID: view.ID, TenantID: tenantID})
	reader, meta, err := audit.DownloadExport(ctx, audit.ExportDeps{Artifacts: deps.Audit.ExportArtifacts, Jobs: deps.Jobs.Repo, Emit: emit}, audit.ExportScope{TenantID: tenantID, RequestedBy: "owner", System: true}, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()
	if _, err := deps.IdManagement.CSVArtifacts.PutCSVArtifact(ctx, tenantID, func(w io.Writer) error { _, err := io.WriteString(w, "abc"); return err }); err != nil {
		t.Fatal(err)
	}
	if err := RunRetentionSweepOnce(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	usage, err := deps.Tenancy.QuotaRepo.GetUsage(ctx, tenantID)
	if err != nil || usage.ExportArtifactsBytes != int(meta.ByteSize)+3 {
		t.Fatalf("usage=%+v err=%v", usage, err)
	}
	if err := RunRetentionSweepOnce(ctx, deps, meta.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	for _, eventType := range []string{"AuditExportRequested", "AuditExportStarted", "AuditExportSucceeded", "AuditExportDownloaded", "AuditExportExpired"} {
		records, err := deps.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: tenantID, Type: eventType})
		if err != nil || len(records) != 1 || records[0].Payload["scope"] != "system" || records[0].Payload["actorUserId"] != "owner" {
			t.Fatalf("%s records=%v err=%v", eventType, records, err)
		}
	}
	for _, terminal := range []string{"failed", "canceled"} {
		view, err := audit.StartExport(ctx, audit.ExportDeps{Artifacts: deps.Audit.ExportArtifacts, Jobs: deps.Jobs.Repo, Enqueuer: deps.Jobs.Enqueuer, Emit: emit}, auditports.ExportParams{Query: auditports.AuditEventQuery{TenantID: tenantID, AllTenants: true}, RequestedBy: "owner", Format: "json", Policy: auditdomain.DefaultExportPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		if terminal == "failed" {
			claimed, err := deps.Jobs.Repo.ClaimBatch(ctx, "worker", jobsdomain.LaneBulk, 1, time.Minute, time.Now().UTC())
			if err != nil || len(claimed) != 1 {
				t.Fatalf("claim=%v err=%v", claimed, err)
			}
			if _, err := deps.Jobs.Repo.Fail(ctx, view.ID, "worker", jobsports.FailOutcome{NextStatus: jobsdomain.StatusFailed, Error: "audit_export_failed"}, time.Now().UTC()); err != nil {
				t.Fatal(err)
			}
			emit(&jobsdomain.JobFailed{At: now, JobID: view.ID, TenantID: tenantID, Terminal: true})
		} else {
			if _, err := deps.Jobs.Repo.Cancel(ctx, view.ID, now); err != nil {
				t.Fatal(err)
			}
			emit(&jobsdomain.JobCanceled{At: now, JobID: view.ID, TenantID: tenantID})
		}
	}
	for _, eventType := range []string{"AuditExportFailed", "AuditExportCanceled"} {
		records, err := deps.Audit.AuditEventRepo.List(ctx, auditports.AuditEventQuery{TenantID: tenantID, Type: eventType})
		if err != nil || len(records) != 1 {
			t.Fatalf("%s records=%v err=%v", eventType, records, err)
		}
	}
}

//spec:covers REQ-AUDIT-009: 環境設定の正の値を配線し、ゼロと不正値を起動時に拒否する。
func TestAuditExportPolicyConfiguration(t *testing.T) {
	cfg := loadSharedConfigOrFatal(t, map[string]string{"AUDIT_EXPORT_MAX_EVENTS": "17", "AUDIT_EXPORT_MAX_BYTES": "2048", "AUDIT_EXPORT_TTL": "2h"})
	if cfg.AuditExportPolicy.MaxEvents != 17 || cfg.AuditExportPolicy.MaxBytes != 2048 || cfg.AuditExportPolicy.TTL != 2*time.Hour {
		t.Fatalf("policy=%+v", cfg.AuditExportPolicy)
	}
	for _, key := range []string{"AUDIT_EXPORT_MAX_EVENTS", "AUDIT_EXPORT_MAX_BYTES", "AUDIT_EXPORT_TTL"} {
		for _, value := range []string{"0", "-1", "invalid"} {
			loader := NewConfigLoader(stubEnv(map[string]string{key: value}))
			LoadSharedConfig(loader)
			if loader.Err() == nil {
				t.Fatalf("accepted %s=%s", key, value)
			}
		}
	}
}
