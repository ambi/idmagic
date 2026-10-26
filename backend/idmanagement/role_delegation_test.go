package idmanagement_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//spec:covers REQ-IDMANAGEMENT-092, REQ-SOURCING-005: 上流属性の変更と直接同期は所属を更新するが、未承認の業務ロールを有効にしない。
func TestAutomaticMembershipNeedsRoleDelegation(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		name := "SCIM membership"
		if dynamic {
			name = "upstream attribute"
		}
		t.Run(name, func(t *testing.T) {
			ctx := testing_tenant.Default(context.Background())
			m := idmanagement.NewMemoryModule()
			now := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
			commands := m.SourceCommands(idmanagement.LifecycleDeps{AttrSchemaRepo: m.AttrSchemaRepo}, "scim", nil)
			group, err := commands.CreateGroup(ctx, &groupdomain.Group{
				TenantID: tenancydomain.DefaultTenantID, Name: "sales", CreatedAt: now,
			})
			if err != nil {
				t.Fatal(err)
			}
			if dynamic {
				group.ID = "dynamic-sales"
				group.Name = "dynamic-sales"
				group.MembershipType = groupdomain.GroupMembershipDynamic
				group.UpstreamManaged = false
			}
			group.Roles = []string{"crm:read"}
			if err := m.GroupRepo.Save(ctx, group); err != nil {
				t.Fatal(err)
			}
			if dynamic {
				if err := m.GroupRepo.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{
					TenantID: group.TenantID, GroupID: group.ID, Expression: `user.department == "Sales"`,
					ReferencedAttributes: []string{"department"}, Version: 1, Enabled: true, CreatedAt: now, UpdatedAt: now,
				}); err != nil {
					t.Fatal(err)
				}
			}
			user, err := commands.CreateUser(ctx, &userdomain.User{
				TenantID: group.TenantID, PreferredUsername: "alice", CreatedAt: now,
				Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: new("Engineering")}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if dynamic {
				user.Attributes["department"] = userdomain.AttributeValue{Type: idmdomain.AttributeTypeString, String: new("Sales")}
				if _, err := commands.UpdateUser(ctx, user); err != nil {
					t.Fatal(err)
				}
			} else if _, err := commands.SetGroupMembership(ctx, group.TenantID, group.ID, user.ID, true, now); err != nil {
				t.Fatal(err)
			}
			memberships, err := m.GroupRepo.ListGroupsByUser(ctx, group.TenantID, user.ID)
			if err != nil || len(memberships) != 1 {
				t.Fatalf("memberships=%+v, err=%v; want one effective membership", memberships, err)
			}
			if roles := groupdomain.EffectiveRoles(user.Roles, memberships); slices.Contains(roles, "crm:read") {
				t.Fatalf("upstream granted an unapproved role: %v", roles)
			}
			expression := ""
			if dynamic {
				expression = `user.department == "Sales"`
			}
			delegation := &groupdomain.GroupRoleDelegation{Roles: []string{"crm:read"}, Expression: expression, ApprovedBy: "admin", ApprovedAt: now}
			if err := m.GroupRepo.SetRoleDelegation(ctx, group.TenantID, group.ID, delegation, now); err != nil {
				t.Fatal(err)
			}
			assertRoles := func(want []string) {
				t.Helper()
				groups, err := m.GroupRepo.ListGroupsByUser(ctx, group.TenantID, user.ID)
				if err != nil || len(groups) != 1 {
					t.Fatalf("membership disappeared: %+v, %v", groups, err)
				}
				if got := groupdomain.EffectiveRoles(nil, groups); !slices.Equal(got, want) {
					t.Fatalf("roles=%v, want %v", got, want)
				}
			}
			assertRoles([]string{"crm:read"})
			if dynamic {
				expression = `user.department != ""`
				if err := m.GroupRepo.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{
					TenantID: group.TenantID, GroupID: group.ID, Expression: expression,
					ReferencedAttributes: []string{"department"}, Version: 2, Enabled: true, CreatedAt: now, UpdatedAt: now,
				}); err != nil {
					t.Fatal(err)
				}
				user.Name = new("Alice updated")
				if _, err := commands.UpdateUser(ctx, user); err != nil {
					t.Fatal(err)
				}
				assertRoles(nil)
			}
			if err := m.GroupRepo.SetRoleDelegation(ctx, group.TenantID, group.ID, nil, now); err != nil {
				t.Fatal(err)
			}
			// 古い Group の通常の保存では、解除した承認を戻せない。
			group.RoleDelegation = delegation
			if err := m.GroupRepo.Save(ctx, group); err != nil {
				t.Fatal(err)
			}
			assertRoles(nil)
			// 移行前の不正な値が保存済みでも、管理権限を認可へ流さない。
			group.Roles = []string{"admin", "crm:read", "system_admin"}
			if err := m.GroupRepo.Save(ctx, group); err != nil {
				t.Fatal(err)
			}
			if err := m.GroupRepo.SetRoleDelegation(ctx, group.TenantID, group.ID, &groupdomain.GroupRoleDelegation{
				Roles: group.Roles, Expression: expression, ApprovedBy: "admin", ApprovedAt: now,
			}, now); err != nil {
				t.Fatal(err)
			}
			assertRoles([]string{"crm:read"})
		})
	}
}
