package db_postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/audit"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	grouppostgres "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_postgres"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type importOwnershipGuard struct{}

func (importOwnershipGuard) SourceManagedUserIDs(context.Context, string, []string) (map[string]bool, error) {
	return map[string]bool{}, nil
}

// TestUserImportApplyRecordsUserCreatedAuditEvent は、CSV インポート apply が発行した UserCreated が、
// PostgreSQL の取り込みの確定と同じトランザクションで AuditEventRepo に記録されることを確かめる。
//
//spec:covers EX-AUDIT-002-01: worker プロセスが実行する CSV インポート apply が発行した UserCreated が、idmagic-api / idmagic-worker のどちらが発行したかにかかわらず同じ AuditEventRepository へ書き込まれ、type=UserCreated の絞り込みで見えることを固定する。
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
	if err := tenancypostgres.NewPostgresModule(db).TenantRepo.Save(ctx, tenant); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	auditRepo := audit.NewPostgresModule(db).AuditEventRepo
	userRepo := &UserRepository{Pool: db}
	planDeps := userusecases.UserImportPlanDeps{
		UserRepo: userRepo, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, OwnershipGuard: importOwnershipGuard{},
	}
	result, err := userusecases.ApplyUserImport(tenantports.WithTenant(ctx, tenant, "", ""), userusecases.UserImportApplyDeps{
		Plan: planDeps, Committer: NewUserImportCommitter(db, tenancypostgres.QuotaRepositoryInTx, audit.RecordWriterInTx, authentication.PasswordHistoryWriterInTx), PasswordHasher: testing_passwords.NewHasher(),
		DynamicGroups: groupusecases.DynamicGroupDeps{GroupRepo: &grouppostgres.GroupRepository{Pool: db}, UserRepo: userRepo},
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
