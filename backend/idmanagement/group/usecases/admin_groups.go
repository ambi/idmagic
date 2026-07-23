package usecases

// 管理者向け Group ライフサイクル操作と user-group membership (ADR-038)。
// SCL IdManagement bounded context が所有する admin インターフェース群:
// ListGroups / GetGroup / CreateGroup / UpdateGroup / DeleteGroup /
// AddGroupMember / RemoveGroupMember / ListUserGroups。
//
// すべての操作は tenancy.TenantID(ctx) のテナント境界に閉じ、cross-tenant な
// 参照・所属は reject する。effective_roles = union(user.roles, group.roles)。

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

var (
	ErrGroupNotFound            = errors.New("group not found")
	ErrGroupNameConflict        = errors.New("group name already exists")
	ErrGroupNameEmpty           = errors.New("group name is required")
	ErrDynamicMembershipManaged = errors.New("dynamic membership is managed by rule")
)

type AdminGroupDeps struct {
	GroupRepo groupports.GroupRepository
	UserRepo  userports.UserRepository
	Emit      func(spec.DomainEvent) error
	// QuotaRepo enforces the tenant's Hard Quota on groups (wi-160, ADR-134).
	// nil skips enforcement (e.g. wiring gaps in tests/tools not yet updated);
	// production bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
}

// GroupView は一覧・詳細でグループとメンバー数をまとめて返す。
type GroupView struct {
	Group       *groupdomain.Group
	MemberCount int
}

func ListGroups(ctx context.Context, deps AdminGroupDeps) ([]GroupView, error) {
	tenantID := tenancy.TenantID(ctx)
	groups, err := deps.GroupRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	views := make([]GroupView, 0, len(groups))
	for _, group := range groups {
		count, err := deps.GroupRepo.CountMembers(ctx, tenantID, group.ID)
		if err != nil {
			return nil, err
		}
		views = append(views, GroupView{Group: group, MemberCount: count})
	}
	return views, nil
}

// GetGroup はグループ本体と所属メンバー一覧を返す。別テナントのグループは
// 未存在として扱う。
func GetGroup(ctx context.Context, deps AdminGroupDeps, id string) (*groupdomain.Group, []*groupdomain.GroupMember, error) {
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	if group == nil {
		return nil, nil, ErrGroupNotFound
	}
	members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	return group, members, nil
}

type CreateGroupInput struct {
	ActorUserID    string
	Name           string
	Description    *string
	Roles          []string
	MembershipType groupdomain.GroupMembershipType
	Now            time.Time
}

func CreateGroup(ctx context.Context, deps AdminGroupDeps, in CreateGroupInput) (*groupdomain.Group, error) {
	tenantID := tenancy.TenantID(ctx)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, ErrGroupNameEmpty
	}
	if err := ensureGroupNameAvailable(ctx, deps, tenantID, name, ""); err != nil {
		return nil, err
	}
	roles, err := idmusecases.NormalizeRoles(in.Roles)
	if err != nil {
		return nil, err
	}
	if err := idmusecases.CheckQuotaAndAudit(ctx, deps.QuotaRepo, deps.Emit, tenantID, tenancydomain.ResourceGroups, idmusecases.NormalizedNow(in.Now)); err != nil {
		return nil, err
	}
	id, err := groupdomain.NewGroupID()
	if err != nil {
		return nil, err
	}
	now := idmusecases.NormalizedNow(in.Now)
	group := &groupdomain.Group{
		ID: id, TenantID: tenantID, Name: name, Description: idmusecases.NormalizeDescription(in.Description),
		Roles: roles, MembershipType: in.MembershipType.Effective(), CreatedAt: now, UpdatedAt: now,
	}
	if err := group.Validate(); err != nil {
		return nil, err
	}
	if err := deps.GroupRepo.Save(ctx, group); err != nil {
		return nil, err
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupCreated{At: now, TenantID: group.TenantID, ActorUserID: in.ActorUserID, GroupID: group.ID}); err != nil {
		return nil, err
	}
	return group, nil
}

type UpdateGroupInput struct {
	ActorUserID string
	ID          string
	Name        *string
	Description *string
	Roles       *[]string
	Now         time.Time
}

func UpdateGroup(ctx context.Context, deps AdminGroupDeps, in UpdateGroupInput) (*groupdomain.Group, error) {
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, in.ID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, ErrGroupNotFound
	}
	updated := *group
	changed := []string{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, ErrGroupNameEmpty
		}
		if name != group.Name {
			if err := ensureGroupNameAvailable(ctx, deps, tenantID, name, group.ID); err != nil {
				return nil, err
			}
			updated.Name = name
			changed = append(changed, "name")
		}
	}
	if in.Description != nil {
		desc := idmusecases.NormalizeDescription(in.Description)
		if !idmusecases.EqualOptionalString(group.Description, desc) {
			updated.Description = desc
			changed = append(changed, "description")
		}
	}
	if in.Roles != nil {
		roles, err := idmusecases.NormalizeRoles(*in.Roles)
		if err != nil {
			return nil, err
		}
		if !slices.Equal(roles, group.Roles) {
			updated.Roles = roles
			changed = append(changed, "roles")
		}
	}
	if len(changed) == 0 {
		return &updated, nil
	}
	now := idmusecases.NormalizedNow(in.Now)
	updated.UpdatedAt = now
	if err := updated.Validate(); err != nil {
		return nil, err
	}
	if err := deps.GroupRepo.Save(ctx, &updated); err != nil {
		return nil, err
	}
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupUpdated{
		At: now, TenantID: group.TenantID, ActorUserID: in.ActorUserID, GroupID: group.ID, ChangedFields: changed,
	}); err != nil {
		return nil, err
	}
	return &updated, nil
}

// DeleteGroup はグループを物理削除し、所属 membership を cascade で解除する。
// 解除メンバーごとに GroupMemberRemoved を emit し、最後に GroupDeleted を emit する。
func DeleteGroup(ctx context.Context, deps AdminGroupDeps, actorUserID, id string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, id)
	if err != nil {
		return err
	}
	now = idmusecases.NormalizedNow(now)
	for _, member := range members {
		removed, err := deps.GroupRepo.RemoveMember(ctx, tenantID, id, member.UserID)
		if err != nil {
			return err
		}
		if removed {
			if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupMemberRemoved{
				At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: id, UserID: member.UserID,
			}); err != nil {
				return err
			}
		}
	}
	if err := deps.GroupRepo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	if deps.QuotaRepo != nil {
		if err := tenancyusecases.DecrementQuota(ctx, deps.QuotaRepo, tenantID, tenancydomain.ResourceGroups, 1); err != nil {
			return err
		}
	}
	return idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupDeleted{At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: id})
}

// AddMember は同一テナントの User をグループに所属させる。既所属なら no-op で
// イベントも emit しない (冪等)。
func AddMember(ctx context.Context, deps AdminGroupDeps, actorUserID, groupID, userID string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	if group.MembershipType.Effective() == groupdomain.GroupMembershipDynamic {
		return ErrDynamicMembershipManaged
	}
	user, err := deps.UserRepo.FindBySub(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil || user.TenantID != tenantID {
		return idmusecases.ErrUserNotFound
	}
	now = idmusecases.NormalizedNow(now)
	added, err := deps.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{
		GroupID: groupID, UserID: userID, Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if added {
		return idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupMemberAdded{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, UserID: userID,
		})
	}
	return nil
}

// RemoveMember はグループから User を外す。非所属なら no-op で event も emit しない。
func RemoveMember(ctx context.Context, deps AdminGroupDeps, actorUserID, groupID, userID string, now time.Time) error {
	tenantID := tenancy.TenantID(ctx)
	group, err := deps.GroupRepo.FindByID(ctx, tenantID, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return ErrGroupNotFound
	}
	if group.MembershipType.Effective() == groupdomain.GroupMembershipDynamic {
		return ErrDynamicMembershipManaged
	}
	now = idmusecases.NormalizedNow(now)
	removed, err := deps.GroupRepo.RemoveMember(ctx, tenantID, groupID, userID)
	if err != nil {
		return err
	}
	if removed {
		return idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupMemberRemoved{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, UserID: userID,
		})
	}
	return nil
}

// UserGroupView は ListUserGroups の結果。明示ロール・グループ由来ロール・union を
// 分けて返し、管理 UI が effective roles を理解しやすくする。
type UserGroupView struct {
	Groups         []*groupdomain.Group
	DirectRoles    []string
	GroupRoles     []string
	EffectiveRoles []string
}

func UserGroups(ctx context.Context, deps AdminGroupDeps, sub string) (*UserGroupView, error) {
	tenantID := tenancy.TenantID(ctx)
	user, err := deps.UserRepo.FindBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenantID {
		return nil, idmusecases.ErrUserNotFound
	}
	groups, err := deps.GroupRepo.ListGroupsByUser(ctx, tenantID, sub)
	if err != nil {
		return nil, err
	}
	directRoles := groupdomain.EffectiveRoles(user.Roles, nil)
	groupRoles := groupdomain.EffectiveRoles(nil, groups)
	return &UserGroupView{
		Groups:         groups,
		DirectRoles:    directRoles,
		GroupRoles:     groupRoles,
		EffectiveRoles: groupdomain.EffectiveRoles(user.Roles, groups),
	}, nil
}

func ensureGroupNameAvailable(ctx context.Context, deps AdminGroupDeps, tenantID, name, excludeID string) error {
	groups, err := deps.GroupRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if group.ID != excludeID && strings.EqualFold(group.Name, name) {
			return ErrGroupNameConflict
		}
	}
	return nil
}
