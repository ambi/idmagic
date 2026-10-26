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

// SourceCommands が User の作成と変更を拒否する理由。取り込み元は、これを項目ごとの失敗へ写す。
// 使用量の超過は、IsQuotaExceeded を持つエラーとして返す。
var (
	ErrSourceUsernameConflict = userusecases.ErrUsernameConflict
	ErrSourceEmailConflict    = userusecases.ErrEmailTaken
	ErrSourceInvalidAttribute = userusecases.ErrInvalidAttribute
)

// SourceCommands は取り込み元が投影した属性を、管理操作と同じ規則と作用で保存する。
// パスワードとロールは取り込み元に任せず、取り込み元の表現と対応表は受け取らない。
type SourceCommands struct {
	users  userusecases.AdminUserDeps
	groups groupusecases.AdminGroupDeps
	actor  string
}

// SourceCommands は、actor の名前でイベントを記録する取り込み元の書き込みを組み立てる。
func (m Module) SourceCommands(d LifecycleDeps, actor string, emit func(spec.DomainEvent) error) SourceCommands {
	users := m.adminUserDeps(d, emit)
	users.UserMutationCommitter = m.UserMutationCommitter
	return SourceCommands{users: users, actor: actor, groups: groupusecases.AdminGroupDeps{
		GroupRepo: m.GroupRepo, UserRepo: m.UserRepo, GroupAttrSchemaRepo: m.GroupAttrSchemaRepo,
		QuotaRepo: d.QuotaRepo, Emit: emit, ProvisioningNotifier: m.GroupProvisioningNotifier,
	}}
}

func (c SourceCommands) CreateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error) {
	return userusecases.CreateSourceUser(tenantports.WithTenantID(ctx, user.TenantID), c.users, c.actor, user)
}

func (c SourceCommands) UpdateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error) {
	return userusecases.UpdateSourceUser(tenantports.WithTenantID(ctx, user.TenantID), c.users, c.actor, user)
}

func (c SourceCommands) CreateGroup(ctx context.Context, group *groupdomain.Group) (*groupdomain.Group, error) {
	return groupusecases.CreateGroup(tenantports.WithTenantID(ctx, group.TenantID), c.groups, groupusecases.CreateGroupInput{
		ActorUserID: c.actor, Name: group.Name, UpstreamManaged: true, Now: group.CreatedAt,
	})
}

func (c SourceCommands) UpdateGroup(ctx context.Context, group *groupdomain.Group) (*groupdomain.Group, error) {
	return groupusecases.UpdateGroup(tenantports.WithTenantID(ctx, group.TenantID), c.groups, groupusecases.UpdateGroupInput{
		ActorUserID: c.actor, ID: group.ID, Name: &group.Name, Now: group.UpdatedAt,
	})
}

func (c SourceCommands) SetGroupMembership(ctx context.Context, tenantID, groupID, userID string, present bool, now time.Time) (bool, error) {
	return (groupusecases.GroupMembershipCommands{Deps: c.groups, Actor: c.actor}).SetGroupMembership(ctx, tenantID, groupID, userID, present, now)
}

func (c SourceCommands) DeleteGroup(ctx context.Context, tenantID, groupID string, now time.Time) error {
	return groupusecases.DeleteGroup(tenantports.WithTenantID(ctx, tenantID), c.groups, c.actor, groupID, now)
}
