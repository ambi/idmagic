package db_postgres

import (
	"context"
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
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

	list, err := repo.ListAll(ctx, tenant.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list groups: %v len=%d", err, len(list))
	}

	added, err := repo.AddMember(ctx, &groupdomain.GroupMember{
		GroupID: group.ID, UserID: user.ID, CreatedAt: testClock(),
	})
	if err != nil || !added {
		t.Fatalf("add member: %v added=%v", err, added)
	}

	members, err := repo.ListMembersByGroup(ctx, tenant.ID, group.ID)
	if err != nil || len(members) != 1 || members[0].UserID != user.ID || members[0].Source != groupdomain.MembershipSourceManual {
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

func TestGroupRepositoryListPage(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedTenant(t, db)
	repo := &GroupRepository{Pool: db}
	ctx := context.Background()
	now := testClock()

	for _, name := range []string{"Charlie", "Alpha", "Bravo", "Delta", "Echo"} {
		g := &groupdomain.Group{
			ID: newUUID(t), TenantID: tenant.ID, Name: name, Roles: []string{},
			CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Save(ctx, g); err != nil {
			t.Fatalf("save %s: %v", name, err)
		}
	}

	first, err := repo.ListPage(ctx, tenant.ID, "", "", 2)
	if err != nil {
		t.Fatalf("list page 1: %v", err)
	}
	if len(first) != 2 || first[0].Name != "Alpha" || first[1].Name != "Bravo" {
		t.Fatalf("unexpected first page: %+v", first)
	}

	last := first[len(first)-1]
	next, err := repo.ListPage(ctx, tenant.ID, last.Name, last.ID, 2)
	if err != nil {
		t.Fatalf("list page 2: %v", err)
	}
	if len(next) != 2 || next[0].Name != "Charlie" || next[1].Name != "Delta" {
		t.Fatalf("unexpected continuation page: %+v", next)
	}

	all, err := repo.ListPage(ctx, tenant.ID, "", "", 100)
	if err != nil {
		t.Fatalf("list page all: %v", err)
	}
	if len(all) != 5 {
		t.Fatalf("expected 5, got %d", len(all))
	}
}
