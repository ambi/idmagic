package domain_test

import (
	"slices"
	"testing"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

//spec:covers REQ-IDMANAGEMENT-092: 自動所属の出力に管理ロールが混ざらず、承認と異なる式ではロールを出さず、元の設定を壊さない。
func FuzzAutomaticMembershipRoleDelegation(f *testing.F) {
	f.Add("catalog:read", `user.department == "Sales"`, false)
	f.Add("admin", "", true)
	f.Add("system_admin", "日本語", false)
	f.Fuzz(func(t *testing.T, businessRole, expression string, dynamic bool) {
		roles := []string{"admin", businessRole, "system_admin"}
		g := groupdomain.Group{Roles: roles, UpstreamManaged: !dynamic, RoleDelegation: &groupdomain.GroupRoleDelegation{Roles: slices.Clone(roles), Expression: expression}}
		if dynamic {
			g.MembershipType = groupdomain.GroupMembershipDynamic
		}
		before := slices.Clone(roles)
		granted := g.RolesForMembership(expression)
		if slices.Contains(granted, "admin") || slices.Contains(granted, "system_admin") {
			t.Fatalf("administrative role escaped: %v", granted)
		}
		if businessRole != "admin" && businessRole != "system_admin" && !slices.Contains(granted, businessRole) {
			t.Fatalf("approved business role missing: %v", granted)
		}
		if got := g.RolesForMembership(expression + " "); len(got) != 0 {
			t.Fatalf("changed expression granted %v", got)
		}
		if len(granted) > 0 {
			granted[0] = "changed"
		}
		if !slices.Equal(g.Roles, before) || !slices.Equal(g.RoleDelegation.Roles, before) {
			t.Fatal("projection changed configuration")
		}
	})
}
