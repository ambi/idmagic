package deps_http

import (
	"context"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// Principals は、共有の HTTP 支援の認可へ、User を主体として、Group 由来のロールを合成して渡す。
// Groups がなければ直接のロールだけを使う。
type Principals struct {
	Users  userports.UserRepository
	Groups groupports.GroupRepository
}

// FindPrincipal は sub の User を主体として返す。Roles は直接のロールである。
func (p Principals) FindPrincipal(ctx context.Context, subject string) (*support.Principal, error) {
	user, err := p.Users.FindBySub(ctx, subject)
	if err != nil || user == nil {
		return nil, err
	}
	return &support.Principal{ID: user.ID, TenantID: user.TenantID, Roles: user.Roles, Active: user.IsActive()}, nil
}

// EffectiveRoles は直接のロールに Group 由来のロールを合成する。所属を引けなければ
// 直接のロールだけを返し、Group 由来の権限を与えない。
func (p Principals) EffectiveRoles(ctx context.Context, principal support.Principal) []string {
	if p.Groups == nil {
		return principal.Roles
	}
	groups, err := p.Groups.ListGroupsByUser(ctx, principal.TenantID, principal.ID)
	if err != nil {
		return principal.Roles
	}
	return groupdomain.EffectiveRoles(principal.Roles, groups)
}
