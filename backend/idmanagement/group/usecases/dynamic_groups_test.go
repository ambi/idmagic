package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestDynamicGroupRuleReconcilesMembership(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
	groups := groupmemory.NewGroupRepository()
	users := usermemory.NewUserRepository()
	now := time.Now().UTC()
	department := "Engineering"
	group := &groupdomain.Group{ID: "g1", TenantID: "acme", Name: "engineering", MembershipType: groupdomain.GroupMembershipDynamic, Roles: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := groups.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	user := &userdomain.User{ID: "u1", TenantID: "acme", PreferredUsername: "alice", PasswordHash: "x", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]userdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	deps := groupusecases.DynamicGroupDeps{GroupRepo: groups, UserRepo: users}
	rule, err := groupusecases.UpdateDynamicGroupRule(ctx, deps, "admin", "g1", `user.department == "Engineering"`, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = groupusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", "g1", true, now); err != nil {
		t.Fatal(err)
	}
	members, _ := groups.ListMembersByGroup(ctx, "acme", "g1")
	if len(members) != 1 || members[0].Source != groupdomain.MembershipSourceDynamicRule || members[0].RuleVersion == nil || *members[0].RuleVersion == rule.Version {
		t.Fatalf("unexpected members: %+v", members)
	}
	if err := groupusecases.AddMember(ctx, groupusecases.AdminGroupDeps{GroupRepo: groups, UserRepo: users}, "admin", "g1", "u1", now); !errors.Is(err, groupusecases.ErrDynamicMembershipManaged) {
		t.Fatalf("manual add err=%v", err)
	}
}
