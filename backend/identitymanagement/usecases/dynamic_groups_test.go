package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/identitymanagement/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestDynamicGroupRuleReconcilesMembership(t *testing.T) {
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
	groups := idmmemory.NewGroupRepository()
	users := idmmemory.NewUserRepository()
	now := time.Now().UTC()
	department := "Engineering"
	group := &idmdomain.Group{ID: "g1", TenantID: "acme", Name: "engineering", MembershipType: idmdomain.GroupMembershipDynamic, Roles: []string{}, CreatedAt: now, UpdatedAt: now}
	if err := groups.Save(ctx, group); err != nil {
		t.Fatal(err)
	}
	user := &idmdomain.User{ID: "u1", TenantID: "acme", PreferredUsername: "alice", PasswordHash: "x", Lifecycle: idmdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, Attributes: map[string]idmdomain.AttributeValue{"department": {Type: idmdomain.AttributeTypeString, String: &department}}, CreatedAt: now, UpdatedAt: now}
	if err := users.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	deps := idmusecases.DynamicGroupDeps{GroupRepo: groups, UserRepo: users}
	rule, err := idmusecases.UpdateDynamicGroupRule(ctx, deps, "admin", "g1", `user.department == "Engineering"`, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = idmusecases.SetDynamicGroupRuleEnabled(ctx, deps, "admin", "g1", true, now); err != nil {
		t.Fatal(err)
	}
	members, _ := groups.ListMembersByGroup(ctx, "acme", "g1")
	if len(members) != 1 || members[0].Source != idmdomain.MembershipSourceDynamicRule || members[0].RuleVersion == nil || *members[0].RuleVersion == rule.Version {
		t.Fatalf("unexpected members: %+v", members)
	}
	if err := idmusecases.AddMember(ctx, idmusecases.AdminGroupDeps{GroupRepo: groups, UserRepo: users}, "admin", "g1", "u1", now); !errors.Is(err, idmusecases.ErrDynamicMembershipManaged) {
		t.Fatalf("manual add err=%v", err)
	}
}
