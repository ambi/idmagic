// Package testing_contract defines the shared group persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement/group/domain"
	"github.com/ambi/idmagic/backend/idmanagement/group/ports"
)

type Fixture struct {
	Repository ports.GroupRepository
	TenantID   string
	UserID     string
	Group      *domain.Group
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.Repository.Save(ctx, f.Group); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.FindByID(ctx, f.TenantID, f.Group.ID)
	if err != nil || got == nil || got.Name != f.Group.Name {
		t.Fatalf("FindByID = (%+v, %v)", got, err)
	}
	all, err := f.Repository.ListAll(ctx, f.TenantID)
	if err != nil || len(all) == 0 || all[0].ID != f.Group.ID {
		t.Fatalf("ListAll = (%+v, %v)", all, err)
	}
	if count, err := f.Repository.Count(ctx, f.TenantID); err != nil || count < 1 {
		t.Fatalf("Count = (%d, %v)", count, err)
	}
	member := &domain.GroupMember{GroupID: f.Group.ID, UserID: f.UserID, Source: domain.MembershipSourceManual, CreatedAt: f.Now}
	added, err := f.Repository.AddMember(ctx, member)
	if err != nil || !added {
		t.Fatalf("AddMember = (%v, %v)", added, err)
	}
	if added, err := f.Repository.AddMember(ctx, member); err != nil || added {
		t.Fatalf("duplicate AddMember = (%v, %v)", added, err)
	}
	members, err := f.Repository.ListMembersByGroup(ctx, f.TenantID, f.Group.ID)
	if err != nil || len(members) != 1 || members[0].UserID != f.UserID {
		t.Fatalf("ListMembersByGroup = (%+v, %v)", members, err)
	}
	if removed, err := f.Repository.RemoveMember(ctx, f.TenantID, f.Group.ID, f.UserID); err != nil || !removed {
		t.Fatalf("RemoveMember = (%v, %v)", removed, err)
	}
	if removed, err := f.Repository.RemoveMember(ctx, f.TenantID, f.Group.ID, f.UserID); err != nil || removed {
		t.Fatalf("duplicate RemoveMember = (%v, %v)", removed, err)
	}
	if err := f.Repository.Delete(ctx, f.TenantID, f.Group.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.FindByID(ctx, f.TenantID, f.Group.ID); err != nil || got != nil {
		t.Fatalf("FindByID after delete = (%+v, %v)", got, err)
	}
}
