package db_postgres

import (
	"context"
	"errors"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestCharacterizeUserImportRowQuota(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	quotas := tenancypostgres.NewQuotaRepository(db)

	importRow := func(t *testing.T, tenantID, actorID, username string) (*userdomain.User, error) {
		t.Helper()
		now := testClock()
		imported := &userdomain.User{
			ID: newUUID(t), TenantID: tenantID, PreferredUsername: username, PasswordHash: "imported-hash",
			Roles: []string{}, CreatedAt: now, UpdatedAt: now,
		}
		err := NewUserImportRowCommitter(db, tenancypostgres.QuotaRepositoryInTx).CommitUserImportRow(ctx, userports.UserImportRowMutation{
			After: imported, Changed: []string{"password"}, ActorUserID: actorID, AuditEventType: "user.imported",
			ConsumesUserQuota: true, Now: now,
		})
		return imported, err
	}
	seedQuota := func(t *testing.T, limit int) (tenantID, actorID string) {
		t.Helper()
		tenant := seedTenant(t, db)
		actor := seedUser(t, db, tenant.ID)
		if err := quotas.SetQuota(context.Background(), tenant.ID, &tenancydomain.TenantQuota{Users: &limit}); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
		return tenant.ID, actor.ID
	}

	t.Run("under_limit_saves_the_user_and_counts_it", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 5)
		imported, err := importRow(t, tenantID, actorID, uniqueID("imported"))
		if err != nil {
			t.Fatalf("CommitUserImportRow: %v", err)
		}
		assertUserStored(t, db, imported.ID, true)
		assertUserUsage(t, quotas, tenantID, 1)
	})

	t.Run("at_limit_refuses_without_saving_or_counting", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 0)
		imported, err := importRow(t, tenantID, actorID, uniqueID("imported"))
		var exceeded *tenancydomain.QuotaExceededError
		if !errors.As(err, &exceeded) || exceeded.TenantID != tenantID || exceeded.Resource != tenancydomain.ResourceUsers {
			t.Fatalf("CommitUserImportRow error = %v, want QuotaExceededError for users of %s", err, tenantID)
		}
		assertUserStored(t, db, imported.ID, false)
		assertUserUsage(t, quotas, tenantID, 0)
	})

	t.Run("failed_row_write_rolls_the_count_back", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 5)
		username := uniqueID("imported")
		if _, err := importRow(t, tenantID, actorID, username); err != nil {
			t.Fatalf("first CommitUserImportRow: %v", err)
		}
		duplicate, err := importRow(t, tenantID, actorID, username)
		if err == nil {
			t.Fatal("CommitUserImportRow with a taken username succeeded, want the row write to fail")
		}
		assertUserStored(t, db, duplicate.ID, false)
		assertUserUsage(t, quotas, tenantID, 1)
	})
}

func assertUserStored(t *testing.T, db sharedpg.DB, userID string, want bool) {
	t.Helper()
	got, err := (&UserRepository{Pool: db}).FindBySub(context.Background(), userID)
	if err != nil {
		t.Fatalf("FindBySub(%s): %v", userID, err)
	}
	if (got != nil) != want {
		t.Fatalf("user %s stored = %v, want %v", userID, got != nil, want)
	}
}

func assertUserUsage(t *testing.T, quotas *tenancypostgres.QuotaRepository, tenantID string, want int) {
	t.Helper()
	usage, err := quotas.GetUsage(context.Background(), tenantID)
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if usage.Users != want {
		t.Fatalf("usage.Users = %d, want %d", usage.Users, want)
	}
}
