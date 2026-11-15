package idmanagement

import (
	"context"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// SeedCommands は固定 ID の初期データにも管理操作の検証と作用を適用する。
type SeedCommands struct {
	users  userusecases.AdminUserDeps
	groups groupusecases.AdminGroupDeps
}

func (m Module) SeedCommands(d LifecycleDeps, emit func(spec.DomainEvent) error) SeedCommands {
	users := m.adminUserDeps(d, emit)
	users.UserMutationCommitter = m.UserMutationCommitter
	return SeedCommands{users: users, groups: groupusecases.AdminGroupDeps{
		GroupRepo: m.GroupRepo, UserRepo: m.UserRepo, GroupAttrSchemaRepo: m.GroupAttrSchemaRepo,
		QuotaRepo: d.QuotaRepo, Emit: emit, ProvisioningNotifier: m.GroupProvisioningNotifier,
	}}
}

func (c SeedCommands) CreateUser(ctx context.Context, user *userdomain.User, password string) error {
	return userusecases.CreateSeedUser(tenantports.WithTenantID(ctx, user.TenantID), c.users, user, password)
}

func (c SeedCommands) CreateGroup(ctx context.Context, group *groupdomain.Group) error {
	return groupusecases.CreateSeedGroup(tenantports.WithTenantID(ctx, group.TenantID), c.groups, group)
}

func (c SeedCommands) AddGroupMember(ctx context.Context, tenantID, groupID, userID string, now time.Time) error {
	_, err := (groupusecases.GroupMembershipCommands{Deps: c.groups, Actor: "seed"}).SetGroupMembership(ctx, tenantID, groupID, userID, true, now)
	return err
}
