package db_postgres

import (
	"context"
	"errors"
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestCharacterizeGroupImportRowQuota(t *testing.T) {
	db := pgtest.Require(t)
	ctx := context.Background()
	quotas := tenancypostgres.NewQuotaRepository(db)
	committer := NewGroupImportRowCommitter(db, tenancypostgres.QuotaRepositoryInTx)

	newGroup := func(t *testing.T, tenantID, name string) *groupdomain.Group {
		t.Helper()
		now := testClock()
		return &groupdomain.Group{
			ID: newUUID(t), TenantID: tenantID, Name: name, Roles: []string{}, CreatedAt: now, UpdatedAt: now,
		}
	}
	createRow := func(group *groupdomain.Group, actorID string) error {
		return committer.CommitGroupImportRow(ctx, groupports.GroupImportRowMutation{
			After: group, Changed: []string{"name"}, ActorUserID: actorID, AuditEventType: "group.imported",
			ConsumesGroupQuota: true, Now: testClock(),
		})
	}
	seedQuota := func(t *testing.T, limit int) (tenantID, actorID string) {
		t.Helper()
		tenant := seedTenant(t, db)
		actor := seedUser(t, db, tenant.ID)
		if err := quotas.SetQuota(context.Background(), tenant.ID, &tenancydomain.TenantQuota{Groups: &limit}); err != nil {
			t.Fatalf("SetQuota: %v", err)
		}
		return tenant.ID, actor.ID
	}

	t.Run("under_limit_saves_the_group_and_counts_it", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 5)
		group := newGroup(t, tenantID, uniqueID("imported-group"))
		if err := createRow(group, actorID); err != nil {
			t.Fatalf("CommitGroupImportRow: %v", err)
		}
		assertGroupStored(ctx, t, db, tenantID, group.ID, true)
		assertGroupUsage(ctx, t, quotas, tenantID, 1)
	})

	t.Run("at_limit_refuses_without_saving_or_counting", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 0)
		group := newGroup(t, tenantID, uniqueID("imported-group"))
		err := createRow(group, actorID)
		var exceeded *tenancydomain.QuotaExceededError
		if !errors.As(err, &exceeded) || exceeded.TenantID != tenantID || exceeded.Resource != tenancydomain.ResourceGroups {
			t.Fatalf("CommitGroupImportRow error = %v, want QuotaExceededError for groups of %s", err, tenantID)
		}
		assertGroupStored(ctx, t, db, tenantID, group.ID, false)
		assertGroupUsage(ctx, t, quotas, tenantID, 0)
	})

	t.Run("failed_row_write_rolls_the_count_back", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 5)
		name := uniqueID("imported-group")
		if err := createRow(newGroup(t, tenantID, name), actorID); err != nil {
			t.Fatalf("first CommitGroupImportRow: %v", err)
		}
		duplicate := newGroup(t, tenantID, name)
		if err := createRow(duplicate, actorID); err == nil {
			t.Fatal("CommitGroupImportRow with a taken name succeeded, want the row write to fail")
		}
		assertGroupStored(ctx, t, db, tenantID, duplicate.ID, false)
		assertGroupUsage(ctx, t, quotas, tenantID, 1)
	})

	t.Run("deletion_releases_the_count", func(t *testing.T) {
		tenantID, actorID := seedQuota(t, 5)
		group := newGroup(t, tenantID, uniqueID("imported-group"))
		if err := createRow(group, actorID); err != nil {
			t.Fatalf("CommitGroupImportRow: %v", err)
		}
		err := committer.CommitGroupImportRow(ctx, groupports.GroupImportRowMutation{
			Delete: true, Before: group, ActorUserID: actorID, AuditEventType: "GroupDeleted",
			ReleasesGroupQuota: true, Now: testClock(),
		})
		if err != nil {
			t.Fatalf("CommitGroupImportRow(delete): %v", err)
		}
		assertGroupStored(ctx, t, db, tenantID, group.ID, false)
		assertGroupUsage(ctx, t, quotas, tenantID, 0)
		// 二つの記録は同じ時刻なので、読み出す順序は ID で決まる。種類の集合だけを比べる。
		events := importAuditEvents(ctx, t, tenantID)
		types := map[string]int{}
		for _, event := range events {
			types[event.eventType]++
		}
		if len(events) != 2 || types["group.imported"] != 1 || types["GroupDeleted"] != 1 {
			t.Fatalf("audit events = %+v, want one group.imported and one GroupDeleted", events)
		}
	})
}

func assertGroupStored(ctx context.Context, t *testing.T, db sharedpg.DB, tenantID, groupID string, want bool) {
	t.Helper()
	var count int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM groups WHERE tenant_id = $1 AND id = $2`, tenantID, groupID).
		Scan(&count); err != nil {
		t.Fatalf("query group %s: %v", groupID, err)
	}
	if (count == 1) != want {
		t.Fatalf("group %s stored = %v, want %v", groupID, count == 1, want)
	}
}

func assertGroupUsage(ctx context.Context, t *testing.T, quotas *tenancypostgres.QuotaRepository, tenantID string, want int) {
	t.Helper()
	usage, err := quotas.GetUsage(ctx, tenantID)
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if usage.Groups != want {
		t.Fatalf("usage.Groups = %d, want %d", usage.Groups, want)
	}
}
