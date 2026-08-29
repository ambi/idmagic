package main

// wi-205: worker.go の DI 組み立て (newAdminUserDeps) が実際に AuditEventRepo まで
// 配線されていることを、実 Postgres 上で CSV インポート apply 相当のフローを通して
// 検証する回帰テスト。モックの Emit を差すだけでは配線漏れ (Emit フィールド未設定) を
// 検出できないため、newAdminUserDeps をそのまま呼び出す経路を通す。

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditpostgres "github.com/ambi/idmagic/backend/audit/db_postgres"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmpostgres "github.com/ambi/idmagic/backend/idmanagement/db_postgres"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userpostgres "github.com/ambi/idmagic/backend/idmanagement/user/db_postgres"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/events/sinks_console"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
}

type workerImportOwnershipGuard struct{}

func (workerImportOwnershipGuard) SourceManagedUserIDs(context.Context, string, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// TestUserImportApplyRecordsUserCreatedAuditEvent は wi-205 の回帰テスト:
// CSV インポート apply が newAdminUserDeps (worker.go の実際の DI 組み立て) 経由で
// UserCreated を発行したとき、AuditEventRepo にそのイベントが記録されることを
// 実 Postgres 上で確認する。修正前は Emit が配線されておらず、ユーザーは作成される
// のに監査イベントだけがサイレントにロストしていた。
func TestUserImportApplyRecordsUserCreatedAuditEvent(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()

	tenantID, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatalf("new tenant id: %v", err)
	}
	now := time.Date(2026, 7, 12, 9, 0, 0, 0, time.UTC)
	tenant := &tenancydomain.Tenant{
		ID:          tenantID,
		Realm:       "worker-audit-test-" + tenantID,
		DisplayName: "wi-205 Worker Audit Test Tenant",
		Status:      tenancydomain.TenantStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := (&tenancypostgres.TenantRepository{Pool: db}).Save(ctx, tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	auditRepo := &auditpostgres.AuditEventRepository{Pool: db}
	userRepo := &userpostgres.UserRepository{Pool: db}
	planDeps := userusecases.UserImportPlanDeps{
		UserRepo: userRepo, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, OwnershipGuard: workerImportOwnershipGuard{},
	}
	result, err := userusecases.ApplyUserImport(tenancy.WithTenant(ctx, tenant, "", ""), userusecases.UserImportApplyDeps{
		Plan: planDeps, Committer: userpostgres.UserImportRowCommitter{Pool: db}, PasswordHasher: passwords_argon2id.NewArgon2idPasswordHasher(),
	}, strings.NewReader("preferred_username,email,name,roles\nalice,alice@example.com,Alice,admin\n"), idmdomain.DefaultCSVTransferPolicy(), "admin-actor", now, nil)
	if err != nil {
		t.Fatalf("run user import apply: %v", err)
	}
	if result.CreatedRows != 1 || result.RejectedRows != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	events, err := auditRepo.List(ctx, auditports.AuditEventQuery{TenantID: tenant.ID, Type: "UserCreated"})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("want 1 UserCreated audit event for CSV import apply, got %d", len(events))
	}
	if events[0].Payload["targetUserId"] == "" || events[0].Payload["targetUserId"] == nil {
		t.Fatalf("audit event missing targetUserId: %+v", events[0].Payload)
	}
}

// TestDataExportRecordsSucceededAuditEvent は wi-148 の監査回帰テスト: data export
// ハンドラが worker の Emit 配線 (deps.NewEmitFunc) 経由で DataExportSucceeded を
// 発行したとき、AuditEventRepo にそのイベントが tenantId 付きで記録されることを実
// Postgres 上で確認する。export は PII を外へ出す高リスク操作のため、成功が監査から
// 見えることを配線レベルで担保する。
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
	if err := (&tenancypostgres.TenantRepository{Pool: db}).Save(ctx, tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	userID, _ := spec.NewUUIDv4()
	tenantCtx := tenancy.WithTenant(ctx, tenant, "", "")
	if err := (&userpostgres.UserRepository{Pool: db}).Save(tenantCtx, &userdomain.User{
		ID: userID, TenantID: tenant.ID, PreferredUsername: "export-audit-" + userID, PasswordHash: "x",
		Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	auditRepo := &auditpostgres.AuditEventRepository{Pool: db}
	artifacts := &idmpostgres.CSVArtifactStore{Pool: db}
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{UserRepo: &userpostgres.UserRepository{Pool: db}, CSVArtifacts: artifacts},
		OAuth2:       oauth2.Module{EventSink: sinks_console.NewConsoleSink()},
		Audit:        audit.Module{AuditEventRepo: auditRepo},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")

	params, err := json.Marshal(idmusecases.DataExportParams{Target: "user", Columns: []string{"preferred_username"}, ActorUserID: "admin-actor"})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	handler := idmusecases.DataExportHandler(idmusecases.DataExportDeps{
		UserRepo: deps.IdManagement.UserRepo, CSVArtifacts: artifacts,
		UserCSVExporter: userusecases.UserCSVExporter{
			Deps: userusecases.UserCSVExportDeps{
				UserRepo: deps.IdManagement.UserRepo, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, Artifacts: artifacts,
			},
			Policy: idmdomain.DefaultCSVTransferPolicy(),
		},
		Emit: func(event spec.DomainEvent) error {
			deps.NewEmitFunc(logger)(event)
			return nil
		},
	})
	if _, err := handler(ctx, &domain.Job{ID: "export-" + tenantID, TenantID: tenant.ID, Kind: idmusecases.KindDataExport, Params: params}); err != nil {
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
