package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

// TestDataExportRecordsSucceededAuditEvent は、data export ハンドラーが worker の Emit 配線
// (deps.NewEmitFunc) 経由で DataExportSucceeded を発行したとき、AuditEventRepo にそのイベントが
// tenantId 付きで記録されることを実 Postgres 上で確認する。export は PII を外へ出す高リスク操作のため、
// 成功が監査から見えることを配線レベルで担保する。
//
//spec:covers REQ-IDMANAGEMENT-086: 正式なハンドラー登録と Runner で succeeded を保存した後、成功の監査記録を一件残す。
func TestDataExportRecordsSucceededAuditEvent(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()

	tenantID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new tenant id: %v", err)
	}
	now := time.Date(2026, 7, 25, 9, 0, 0, 0, time.UTC)
	tenant := &tenancydomain.Tenant{
		ID: tenantID, Realm: "export-audit-test-" + tenantID, DisplayName: "wi-148 Export Audit Test Tenant",
		Status: tenancydomain.TenantStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := tenancypostgres.NewPostgresModule(db).TenantRepo.Save(ctx, tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	idmModule := idmanagement.NewPostgresModule(db).WithPostgresImports(db, idmanagement.PostgresImportDeps{
		QuotasInTx:  tenancypostgres.QuotaRepositoryInTx,
		AuditInTx:   audit.RecordWriterInTx,
		HistoryInTx: authentication.PasswordHistoryWriterInTx,
	})
	userID, _ := spec.NewUUIDv4()
	tenantCtx := tenantports.WithTenant(ctx, tenant, "", "")
	if err := idmModule.UserRepo.Save(tenantCtx, &userdomain.User{
		ID: userID, TenantID: tenant.ID, PreferredUsername: "export-audit-" + userID, PasswordHash: "x",
		Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	auditRepo := audit.NewPostgresModule(db).AuditEventRepo
	deps := &bootstrap.Dependencies{
		IdManagement: idmModule,
		OAuth2:       oauth2.Module{EventSink: sinks_console.NewConsoleSink()},
		Audit:        audit.Module{AuditEventRepo: auditRepo},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")

	runCtx, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	jobRepo := jobs.NewPostgresModule(db).Repo
	handlers := jobs.NewHandlerRegistry()
	var exportID string
	deps.IdManagement.RegisterJobHandlers(handlers, idmanagement.JobHandlerDeps{
		EnqueuerInTx:   jobs.NewEnqueuerInTx,
		Jobs:           jobRepo,
		AttrSchemaRepo: idmModule.AttrSchemaRepo,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			if succeeded, ok := event.(*idmdomain.DataExportSucceeded); ok && succeeded.ExportID == exportID {
				stop()
			}
			return nil
		},
	})
	const kindDataExport domain.JobKind = "data_export"
	params, err := json.Marshal(map[string]any{"target": "user", "columns": []string{"preferred_username"}, "actor_user_id": "admin-actor"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	job, _, err := jobRepo.Enqueue(ctx, ports.EnqueueInput{
		TenantID: tenant.ID, Kind: kindDataExport, Lane: domain.LaneBulk, Params: params,
		MaxAttempts: 1, Now: now, RunAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	exportID = job.ID
	runner := jobs.NewRunner(jobs.RunnerConfig{
		WorkerID: "export-audit-worker", Lane: domain.LaneBulk, PollInterval: time.Millisecond,
	}, jobs.RunnerDeps{Repo: jobRepo, Handlers: handlers})
	_ = runner.Run(runCtx)
	stored, err := jobRepo.Get(ctx, job.ID)
	if err != nil || stored.Status != domain.StatusSucceeded {
		t.Fatalf("保存結果=(%+v, %v)", stored, err)
	}

	events, err := auditRepo.List(ctx, auditports.AuditEventQuery{TenantID: tenant.ID, Type: "DataExportSucceeded"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 DataExportSucceeded audit event, got %d", len(events))
	}
	if events[0].Payload["target"] != "user" {
		t.Fatalf("audit event target = %v, want user", events[0].Payload["target"])
	}
}
