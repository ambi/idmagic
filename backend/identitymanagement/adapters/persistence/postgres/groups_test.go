package postgres

import (
	"context"
	"testing"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/persistence/postgres/pgtest"
)

func TestGroupRepositoryRoundTripAndMembers(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	user := seedUser(t, db, tenant.ID)
	repo := &GroupRepository{Pool: db}
	ctx := context.Background()

	group := seedGroup(t, db, tenant.ID)

	got, err := repo.FindByID(ctx, tenant.ID, group.ID)
	if err != nil || got == nil || got.Name != group.Name {
		t.Fatalf("find group: %v %+v", err, got)
	}

	list, err := repo.ListByTenant(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list groups: %v len=%d", err, len(list))
	}

	added, err := repo.AddMember(ctx, &idmdomain.GroupMember{
		GroupID: group.ID, UserID: user.ID, CreatedAt: testClock(),
	})
	if err != nil || !added {
		t.Fatalf("add member: %v added=%v", err, added)
	}

	members, err := repo.ListMembersByGroup(ctx, tenant.ID, group.ID)
	if err != nil || len(members) != 1 || members[0].UserID != user.ID {
		t.Fatalf("list members: %v %+v", err, members)
	}

	count, err := repo.CountMembers(ctx, tenant.ID, group.ID)
	if err != nil || count != 1 {
		t.Fatalf("count members: %v count=%d", err, count)
	}

	groups, err := repo.ListGroupsByUser(ctx, tenant.ID, user.ID)
	if err != nil || len(groups) != 1 {
		t.Fatalf("groups by user: %v len=%d", err, len(groups))
	}

	removed, err := repo.RemoveMember(ctx, tenant.ID, group.ID, user.ID)
	if err != nil || !removed {
		t.Fatalf("remove member: %v removed=%v", err, removed)
	}

	if err := repo.Delete(ctx, tenant.ID, group.ID); err != nil {
		t.Fatalf("delete group: %v", err)
	}
}
