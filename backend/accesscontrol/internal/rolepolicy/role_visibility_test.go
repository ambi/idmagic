package rolepolicy

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-OAUTH2-004: ロールに可視な権限だけを表示し、制御面の能力を区別する。
func TestListRolePoliciesKeepsSystemRolePermissionsSeparate(t *testing.T) {
	for _, test := range []struct {
		name         string
		actorRoles   []string
		controlPlane bool
		visible      bool
	}{
		{name: "system_admin_on_control_plane", actorRoles: []string{"system_admin"}, controlPlane: true, visible: true},
		{name: "system_admin_outside_control_plane", actorRoles: []string{"system_admin"}},
		{name: "admin_on_control_plane", actorRoles: []string{"admin"}, controlPlane: true},
		{name: "no_system_role_on_control_plane", controlPlane: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			roles, err := ListRolePolicies(spec.CurrentRuntimeContract(), test.actorRoles, test.controlPlane)
			if err != nil {
				t.Fatal(err)
			}
			systemAdmin := findRole(t, roles, "system_admin")
			if hasPermission(systemAdmin, "AdminUserRead") {
				t.Fatal("system_admin に admin 専用の AdminUserRead が含まれている")
			}
			for _, permission := range []string{"AdminTenantsManage", "SystemKeyHealthRead"} {
				if got := hasPermission(systemAdmin, permission); got != test.visible {
					t.Fatalf("system_admin の権限 %s の可視性 = %v、期待値 %v", permission, got, test.visible)
				}
			}
			if !test.visible && len(systemAdmin.Permissions) != 0 {
				t.Fatalf("制御面の境界外で system_admin の権限が見えている: %+v", systemAdmin.Permissions)
			}
		})
	}
}
