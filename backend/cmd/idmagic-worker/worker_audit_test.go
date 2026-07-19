package main

// wi-205: worker.go の DI 組み立て (newAdminUserDeps) が実際に AuditEventRepo まで
// 配線されていることを、実 Postgres 上で CSV インポート apply 相当のフローを通して
// 検証する回帰テスト。モックの Emit を差すだけでは配線漏れ (Emit フィールド未設定) を
// 検出できないため、newAdminUserDeps をそのまま呼び出す経路を通す。

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditpostgres "github.com/ambi/idmagic/backend/audit/adapters/persistence/postgres"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	"github.com/ambi/idmagic/backend/authentication"
	authnpostgres "github.com/ambi/idmagic/backend/authentication/adapters/persistence/postgres"
	"github.com/ambi/idmagic/backend/cmd/internal/bootstrap"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmpostgres "github.com/ambi/idmagic/backend/idmanagement/adapters/persistence/postgres"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/ambi/idmagic/backend/shared/adapters/eventsink"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/adapters/persistence/postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestMain(m *testing.M) {
	os.Exit(pgtest.Main(m))
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
	deps := &bootstrap.Dependencies{
		IdManagement: idmanagement.Module{
			UserRepo: &idmpostgres.UserRepository{Pool: db},
		},
		Authentication: authentication.Module{
			PasswordHistoryRepo: &authnpostgres.PasswordHistoryRepository{Pool: db},
		},
		OAuth2: oauth2.Module{
			EventSink: eventsink.NewConsoleSink(),
		},
		Audit: audit.Module{
			AuditEventRepo: auditRepo,
		},
	}
	logger := logging.New(os.Stderr, logging.ParseLevel("error"), "idmagic-worker-test", "test")
	adminDeps := newAdminUserDeps(deps, logger)

	params, err := json.Marshal(userusecases.UserImportParams{
		CSV:         "preferred_username,email,name,roles\nalice,alice@example.com,Alice,admin\n",
		ActorUserID: "admin-actor",
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	job := &domain.Job{
		ID:       "job-" + tenantID,
		TenantID: tenant.ID,
		Kind:     domain.KindUserImportApply,
		Params:   params,
	}

	handler := userusecases.UserImportHandler(adminDeps, true)
	rawResult, err := handler(ctx, job)
	if err != nil {
		t.Fatalf("run user import apply: %v", err)
	}
	var result userusecases.UserImportResult
	if err := json.Unmarshal(rawResult, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.AcceptedRows != 1 || result.RejectedRows != 0 {
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
