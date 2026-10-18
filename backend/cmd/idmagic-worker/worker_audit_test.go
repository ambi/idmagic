package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idmanagement"
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

// capturedHandlers は登録されたジョブのハンドラーを種類ごとに保持する。
type capturedHandlers map[domain.JobKind]ports.Handler

func (c capturedHandlers) Register(kind domain.JobKind, h ports.Handler) { c[kind] = h }

// TestDataExportRecordsSucceededAuditEvent は、data export ハンドラーが worker の Emit 配線
// (deps.NewEmitFunc) 経由で DataExportSucceeded を発行したとき、AuditEventRepo にそのイベントが
// tenantId 付きで記録されることを実 Postgres 上で確認する。export は PII を外へ出す高リスク操作のため、
// 成功が監査から見えることを配線レベルで担保する。
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
	idmModule := idmanagement.NewPostgresModule(db, tenancypostgres.QuotaRepositoryInTx)
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

	handlers := capturedHandlers{}
	deps.IdManagement.RegisterJobHandlers(handlers, idmanagement.JobHandlerDeps{
		EnqueuerInTx:   jobs.NewEnqueuerInTx,
		AttrSchemaRepo: idmModule.AttrSchemaRepo,
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	})
	const kindDataExport domain.JobKind = "data_export"
	handler := handlers[kindDataExport]
	if handler == nil {
		t.Fatalf("no handler registered for %s", kindDataExport)
	}
	params, err := json.Marshal(map[string]any{"target": "user", "columns": []string{"preferred_username"}, "actor_user_id": "admin-actor"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	if _, err := handler(ctx, &domain.Job{ID: "export-" + tenantID, TenantID: tenant.ID, Kind: kindDataExport, Params: params}); err != nil {
		t.Fatalf("run data export: %v", err)
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
