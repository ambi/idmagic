package usecases

// 管理者向け Group ライフサイクル操作と user-group membership。
// SCL IdManagement bounded context が所有する admin インターフェース群:
// ListGroups / GetGroup / CreateGroup / UpdateGroup / DeleteGroup /
// AddGroupMember / RemoveGroupMember / ListUserGroups。
//
// すべての操作は tenancy.TenantID(ctx) のテナント境界に閉じ、cross-tenant な
// 参照・所属は reject する。effective_roles = union(user.roles, group.roles)。

import (
	"context"
	"errors"
	"net/mail"
	"reflect"
	"slices"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
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
	// ErrInvalidEmail is returned when Group.Email does not parse as a mail address.
	ErrInvalidEmail = errors.New("email is not a valid address")
	// ErrInvalidAttribute is returned when Group.Attributes does not conform to the
	// tenant's effective TenantGroupAttributeSchema.
	ErrInvalidAttribute = errors.New("attribute does not conform to schema")
)

type AdminGroupDeps struct {
	GroupRepo groupports.GroupRepository
	UserRepo  userports.UserRepository
	Emit      func(spec.DomainEvent) error
	// QuotaRepo enforces the tenant's Hard Quota on groups (wi-160).
	// nil skips enforcement (e.g. wiring gaps in tests/tools not yet updated);
	// production bootstrap always sets it.
	QuotaRepo tenantports.QuotaRepository
	// GroupAttrSchemaRepo validates Group.Attributes against the tenant's
	// TenantGroupAttributeSchema. nil rejects any non-empty Attributes (there is no
	// builtin catalog to fall back to, unlike User's AttrSchemaRepo).
	GroupAttrSchemaRepo tenantports.TenantGroupAttributeSchemaRepository
	// ProvisioningNotifier reports a committed Group mutation to outbound
	// Provisioning, mirroring the User side. nil means outbound provisioning is
	// not wired. Whether a delivery results is the connection's decision
	// (push_groups and its GroupPushConfig), not this call's.
	ProvisioningNotifier groupports.ProvisioningNotifier
}

// notifyProvisioning reports a committed Group mutation. The notification is
// deliberately not allowed to fail the mutation that already committed: capture
// runs in its own transaction (the User side makes the same choice), so a
// provisioning outage must not roll back an administrator's Group edit.
func notifyProvisioning(
	ctx context.Context,
	deps AdminGroupDeps,
	tenantID, groupID string,
	trigger groupports.ProvisioningTrigger,
	now time.Time,
) error {
	if deps.ProvisioningNotifier == nil {
		return nil
	}
	return deps.ProvisioningNotifier.NotifyGroupMutation(ctx, tenantID, groupID, trigger, now)
}

// normalizeGroupEmail trims and lowercases in.Email, treating an empty/whitespace-only
// value as "no email" (nil). A non-empty value that does not parse as a mail address
// returns ErrInvalidEmail.
func normalizeGroupEmail(email *string) (*string, error) {
	if email == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*email)
	if trimmed == "" {
		return nil, nil
	}
	addr, err := mail.ParseAddress(trimmed)
	if err != nil {
		return nil, ErrInvalidEmail
	}
	lower := strings.ToLower(addr.Address)
	return &lower, nil
}

// effectiveGroupAttributeDefs returns the tenant's Group attribute definitions. Unlike
// User, Group has no builtin catalog, so an unwired repo or an undefined tenant schema
// resolves to an empty definition set rather than falling back to defaults.
func effectiveGroupAttributeDefs(ctx context.Context, repo tenantports.TenantGroupAttributeSchemaRepository, tenantID string) ([]groupdomain.GroupAttributeDef, error) {
	if repo == nil {
		return nil, nil
	}
	schema, err := repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if schema == nil {
		return nil, nil
	}
	return schema.EffectiveDefs(), nil
}

// validateGroupAttributesInput validates attributes against the tenant's effective
// GroupAttributeDef set, including required-attribute enforcement. It is called
// unconditionally on create (so a tenant-defined required attribute is enforced even
// when the caller omits attributes) and only when the caller supplies a replacement
// map on update.
func validateGroupAttributesInput(ctx context.Context, deps AdminGroupDeps, tenantID string, attributes map[string]userdomain.AttributeValue) error {
	defs, err := effectiveGroupAttributeDefs(ctx, deps.GroupAttrSchemaRepo, tenantID)
	if err != nil {
		return err
	}
	if len(defs) == 0 && len(attributes) == 0 {
		return nil
	}
	if err := groupdomain.ValidateGroupAttributes(attributes, defs); err != nil {
		return errors.Join(ErrInvalidAttribute, err)
	}
	return nil
}

// GroupView は一覧・詳細でグループとメンバー数をまとめて返す。
type GroupView struct {
	Group       *groupdomain.Group
	MemberCount int
}

// ListGroups returns up to limit+1 groups after the given keyset (wi-159)
// — callers pass limit+1 to detect whether a next page exists, then
// trim to limit before responding.
func ListGroups(ctx context.Context, deps AdminGroupDeps, afterName, afterID string, limit int) ([]GroupView, error) {
	tenantID := tenancy.TenantID(ctx)
	groups, err := deps.GroupRepo.ListPage(ctx, tenantID, afterName, afterID, limit)
	return groupViews(ctx, deps, tenantID, groups, err)
}

func ListGroupsBefore(ctx context.Context, deps AdminGroupDeps, beforeName, beforeID string, limit int) ([]GroupView, error) {
	tenantID := tenancy.TenantID(ctx)
	groups, err := deps.GroupRepo.ListPageBefore(ctx, tenantID, beforeName, beforeID, limit)
	return groupViews(ctx, deps, tenantID, groups, err)
}

func groupViews(ctx context.Context, deps AdminGroupDeps, tenantID string, groups []*groupdomain.Group, err error) ([]GroupView, error) {
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
	Email          *string
	Attributes     map[string]userdomain.AttributeValue
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
	email, err := normalizeGroupEmail(in.Email)
	if err != nil {
		return nil, err
	}
	if err := validateGroupAttributesInput(ctx, deps, tenantID, in.Attributes); err != nil {
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
		Email: email, Attributes: in.Attributes,
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
	if err := notifyProvisioning(ctx, deps, group.TenantID, group.ID, groupports.ProvisioningGroupCreated, now); err != nil {
		return nil, err
	}
	return group, nil
}

type UpdateGroupInput struct {
	ActorUserID string
	ID          string
	Name        *string
	Description *string
	Email       *string
	// Attributes は指定時に attributes 全体を置換する (実効スキーマで検証)。
	Attributes *map[string]userdomain.AttributeValue
	Roles      *[]string
	Now        time.Time
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
	if in.Email != nil {
		email, err := normalizeGroupEmail(in.Email)
		if err != nil {
			return nil, err
		}
		if !idmusecases.EqualOptionalString(group.Email, email) {
			updated.Email = email
			changed = append(changed, "email")
		}
	}
	if in.Attributes != nil {
		if err := validateGroupAttributesInput(ctx, deps, tenantID, *in.Attributes); err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(group.Attributes, *in.Attributes) {
			updated.Attributes = *in.Attributes
			changed = append(changed, "attributes")
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
	if err := notifyProvisioning(ctx, deps, group.TenantID, group.ID, groupports.ProvisioningGroupChanged, now); err != nil {
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
	if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupDeleted{At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: id}); err != nil {
		return err
	}
	return notifyProvisioning(ctx, deps, tenantID, id, groupports.ProvisioningGroupDeleted, now)
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
		if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupMemberAdded{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, UserID: userID,
		}); err != nil {
			return err
		}
		return notifyProvisioning(ctx, deps, tenantID, groupID, groupports.ProvisioningGroupMembershipChanged, now)
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
		if err := idmusecases.AdminEmit(deps.Emit, &idmdomain.GroupMemberRemoved{
			At: now, TenantID: tenantID, ActorUserID: actorUserID, GroupID: groupID, UserID: userID,
		}); err != nil {
			return err
		}
		return notifyProvisioning(ctx, deps, tenantID, groupID, groupports.ProvisioningGroupMembershipChanged, now)
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
	groups, err := deps.GroupRepo.ListAll(ctx, tenantID)
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
