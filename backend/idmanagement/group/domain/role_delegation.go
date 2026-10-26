package domain

import (
	"errors"
	"slices"
	"time"
)

// GroupRoleDelegation は、グループの現在のロールと式に結び付けた管理者の委任である。
type GroupRoleDelegation struct {
	Roles      []string  `json:"roles"`
	Expression string    `json:"expression"`
	ApprovedBy string    `json:"approved_by"`
	ApprovedAt time.Time `json:"approved_at"`
}

var (
	ErrAutomaticAdminRole      = errors.New("automatic membership cannot grant administrative roles")
	ErrDelegationGroupNotFound = errors.New("delegation group does not exist")
)

func (g Group) AutomaticMembership() bool {
	return g.UpstreamManaged || g.MembershipType.Effective() == GroupMembershipDynamic
}

func (g Group) ValidateAutomaticRoles() error {
	if g.AutomaticMembership() && (slices.Contains(g.Roles, "admin") || slices.Contains(g.Roles, "system_admin")) {
		return ErrAutomaticAdminRole
	}
	return nil
}

// RolesForMembership は、設定を保存した後の並行変更と古い値も認可へ流さない。
// 所属自体は返し、ロールだけを制限するので、アプリケーションの割り当てを変えない。
func (g Group) RolesForMembership(expression string) []string {
	if !g.AutomaticMembership() {
		return slices.Clone(g.Roles)
	}
	delegation := g.RoleDelegation
	if delegation == nil || delegation.Expression != expression || !slices.Equal(delegation.Roles, g.Roles) {
		return []string{}
	}
	return slices.DeleteFunc(slices.Clone(g.Roles), func(role string) bool {
		return role == "admin" || role == "system_admin"
	})
}
