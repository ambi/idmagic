// Package testing_contract defines the shared group persistence contract.
package testing_contract

import (
	"context"
	"errors"
	"slices"
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
	t.Run("role delegation", func(t *testing.T) { roleDelegation(t, newFixture) })
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

func roleDelegation(t *testing.T, newFixture NewFixture) {
	t.Helper()
	for _, dynamic := range []bool{false, true} {
		t.Run(map[bool]string{false: "upstream", true: "dynamic"}[dynamic], func(t *testing.T) {
			f := newFixture(t)
			ctx := context.Background()
			g := f.Group
			g.Roles = []string{"catalog:read"}
			g.UpstreamManaged = !dynamic
			expression := ""
			member := &domain.GroupMember{GroupID: g.ID, UserID: f.UserID, Source: domain.MembershipSourceManual, CreatedAt: f.Now}
			if dynamic {
				g.MembershipType = domain.GroupMembershipDynamic
				expression = `user.department == "Engineering"`
				member.Source, member.RuleVersion = domain.MembershipSourceDynamicRule, new(int64(1))
			}
			if err := f.Repository.Save(ctx, g); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := f.Repository.Delete(ctx, f.TenantID, g.ID); err != nil {
					t.Error(err)
				}
			})
			if dynamic {
				if err := f.Repository.SaveDynamicRule(ctx, &domain.DynamicGroupRule{TenantID: f.TenantID, GroupID: g.ID, Expression: expression, Enabled: true, Version: 1, CreatedAt: f.Now, UpdatedAt: f.Now}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.Repository.AddMember(ctx, member); err != nil {
				t.Fatal(err)
			}
			assertRoles := func(want []string) {
				t.Helper()
				groups, err := f.Repository.ListGroupsByUser(ctx, f.TenantID, f.UserID)
				if err != nil || len(groups) != 1 || !slices.Equal(groups[0].Roles, want) {
					t.Fatalf("membership projection=%+v err=%v, want roles=%v", groups, err, want)
				}
			}
			assertRoles(nil)
			approval := &domain.GroupRoleDelegation{Roles: g.Roles, Expression: expression, ApprovedBy: f.UserID, ApprovedAt: f.Now}
			if err := f.Repository.SetRoleDelegation(ctx, f.TenantID, g.ID, approval, f.Now); err != nil {
				t.Fatal(err)
			}
			assertRoles([]string{"catalog:read"})
			// ロールの変更は、所属を残したまま承認を無効にする。
			g.Roles = []string{"catalog:write"}
			if err := f.Repository.Save(ctx, g); err != nil {
				t.Fatal(err)
			}
			assertRoles(nil)
			g.Roles = []string{"catalog:read"}
			if err := f.Repository.Save(ctx, g); err != nil {
				t.Fatal(err)
			}
			assertRoles(g.Roles)
			if dynamic {
				// 所属の版が一致しても、式が違えば以前の承認は使えない。
				if err := f.Repository.SaveDynamicRule(ctx, &domain.DynamicGroupRule{TenantID: f.TenantID, GroupID: g.ID, Expression: `user.department != ""`, Enabled: true, Version: 1, CreatedAt: f.Now, UpdatedAt: f.Now}); err != nil {
					t.Fatal(err)
				}
				assertRoles(nil)
				if err := f.Repository.SaveDynamicRule(ctx, &domain.DynamicGroupRule{TenantID: f.TenantID, GroupID: g.ID, Expression: expression, Enabled: true, Version: 1, CreatedAt: f.Now, UpdatedAt: f.Now}); err != nil {
					t.Fatal(err)
				}
				assertRoles(g.Roles)
			}
			// 通常の保存は承認を消すことも、上流管理を解除することもできない。
			g.UpstreamManaged = false
			if err := f.Repository.Save(ctx, g); err != nil {
				t.Fatal(err)
			}
			assertRoles([]string{"catalog:read"})
			if err := f.Repository.SetRoleDelegation(ctx, f.TenantID, g.ID, nil, f.Now); err != nil {
				t.Fatal(err)
			}
			g.RoleDelegation = approval
			if err := f.Repository.Save(ctx, g); err != nil {
				t.Fatal(err)
			}
			assertRoles(nil)
			if err := f.Repository.SetRoleDelegation(ctx, "00000000-0000-4000-8000-ffffffffffff", g.ID, approval, f.Now); !errors.Is(err, domain.ErrDelegationGroupNotFound) {
				t.Fatalf("foreign approval err=%v, want missing group", err)
			}
			assertRoles(nil)
		})
	}
}
