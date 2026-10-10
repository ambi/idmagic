package db_postgres

import (
	"context"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
)

func TestUserImportRowCommitterWritesPasswordHistoryAndAudit(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	tenant := seedTenant(t, db)
	actor := seedUser(t, db, tenant.ID)
	now := testClock()
	imported := &userdomain.User{
		ID: newUUID(t), TenantID: tenant.ID, PreferredUsername: uniqueID("imported"), PasswordHash: "imported-hash",
		Roles: []string{}, CreatedAt: now, UpdatedAt: now,
	}
	err := NewUserImportRowCommitter(db, tenancypostgres.QuotaRepositoryInTx).CommitUserImportRow(ctx, userports.UserImportRowMutation{
		After: imported, Changed: []string{"password"}, ActorUserID: actor.ID, AuditEventType: "user.imported",
		PasswordHistoryHash: "history-hash", Now: now,
	})
	if err != nil {
		t.Fatalf("CommitUserImportRow: %v", err)
	}

	var encoded string
	if err := db.QueryRow(ctx, `SELECT encoded FROM password_history WHERE user_id = $1`, imported.ID).Scan(&encoded); err != nil {
		t.Fatalf("query password history: %v", err)
	}
	if encoded != "history-hash" {
		t.Fatalf("password history = %q, want history-hash", encoded)
	}

	var count int
	var eventType, userID string
	if err := db.QueryRow(ctx, `SELECT count(*) OVER (), type, user_id::text FROM audit_events WHERE tenant_id = $1`, tenant.ID).
		Scan(&count, &eventType, &userID); err != nil {
		t.Fatalf("query audit event: %v", err)
	}
	if count != 1 || eventType != "user.imported" || userID != imported.ID {
		t.Fatalf("audit = %d event(s), type %s user %s; want one user.imported for %s", count, eventType, userID, imported.ID)
	}
}
