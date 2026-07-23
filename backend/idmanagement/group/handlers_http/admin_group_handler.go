package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type groupCreateRequest struct {
	Name           string                          `json:"name"`
	Description    *string                         `json:"description"`
	Roles          []string                        `json:"roles"`
	MembershipType groupdomain.GroupMembershipType `json:"membership_type"`
	DynamicRule    *dynamicRuleRequest             `json:"dynamic_rule"`
}

type dynamicRuleRequest struct {
	Expression string `json:"expression"`
}

type dynamicRulePreviewRequest struct {
	Expression string   `json:"expression"`
	UserIDs    []string `json:"user_ids"`
}

type groupUpdateRequest struct {
	Name        *string   `json:"name"`
	Description *string   `json:"description"`
	Roles       *[]string `json:"roles"`
}

type groupSummaryResponse struct {
	ID             string                          `json:"id"`
	TenantID       string                          `json:"tenant_id"`
	Name           string                          `json:"name"`
	Description    *string                         `json:"description,omitempty"`
	Roles          []string                        `json:"roles"`
	MemberCount    int                             `json:"member_count"`
	CreatedAt      time.Time                       `json:"created_at"`
	UpdatedAt      time.Time                       `json:"updated_at"`
	ScimSource     *string                         `json:"scim_source,omitempty"`
	MembershipType groupdomain.GroupMembershipType `json:"membership_type"`
	DynamicRule    *groupdomain.DynamicGroupRule   `json:"dynamic_rule,omitempty"`
}

type groupMemberResponse struct {
	UserID            string                            `json:"user_id"`
	PreferredUsername string                            `json:"preferred_username"`
	Source            groupdomain.GroupMembershipSource `json:"source"`
	RuleVersion       *int64                            `json:"rule_version,omitempty"`
	CreatedAt         time.Time                         `json:"created_at"`
}

type userGroupsResponse struct {
	Groups         []groupSummaryResponse `json:"groups"`
	DirectRoles    []string               `json:"direct_roles"`
	GroupRoles     []string               `json:"group_roles"`
	EffectiveRoles []string               `json:"effective_roles"`
}

func HandleListGroups(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	views, err := groupusecases.ListGroups(c.Request().Context(), adminGroupDeps(d))
	if err != nil {
		return err
	}
	groups := make([]groupSummaryResponse, len(views))
	for i, view := range views {
		groups[i] = toGroupSummaryResponse(view.Group, view.MemberCount)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"groups": groups})
}

func HandleGetGroup(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	group, members, err := groupusecases.GetGroup(c.Request().Context(), adminGroupDeps(d), c.Param("group_id"))
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	res := toGroupSummaryResponse(group, len(members))
	if group.MembershipType.Effective() == groupdomain.GroupMembershipDynamic {
		res.DynamicRule, err = d.GroupRepo.FindDynamicRule(c.Request().Context(), group.TenantID, group.ID)
		if err != nil {
			return err
		}
	}
	if d.ScimRepo != nil {
		ref, _ := d.ScimRepo.FindGroupRefByGroupID(c.Request().Context(), group.TenantID, group.ID)
		if ref != nil {
			src := "SCIM"
			res.ScimSource = &src
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"group":   res,
		"members": toGroupMemberResponses(c.Request().Context(), d, members),
	})
}

func HandleCreateGroup(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input groupCreateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if input.MembershipType.Effective() == groupdomain.GroupMembershipDynamic && input.DynamicRule != nil {
		defs := userdomain.BuiltinUserAttributeDefs()
		if d.AttrSchemaRepo != nil {
			schema, schemaErr := d.AttrSchemaRepo.FindByTenant(c.Request().Context(), actor.TenantID)
			if schemaErr != nil {
				return schemaErr
			}
			if schema != nil {
				defs = schema.EffectiveDefs()
			}
		}
		if _, compileErr := groupdomain.CompileDynamicGroupRule(input.DynamicRule.Expression, defs); compileErr != nil {
			return writeAdminGroupError(c, errors.Join(groupusecases.ErrInvalidDynamicGroupRule, compileErr))
		}
	}
	group, err := groupusecases.CreateGroup(c.Request().Context(), adminGroupDeps(d), groupusecases.CreateGroupInput{
		ActorUserID: actor.ID, Name: input.Name, Description: input.Description, Roles: input.Roles, MembershipType: input.MembershipType, Now: time.Now().UTC(),
	})
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	if input.DynamicRule != nil && group.MembershipType.Effective() == groupdomain.GroupMembershipDynamic {
		if _, err := groupusecases.UpdateDynamicGroupRule(c.Request().Context(), dynamicGroupDeps(d), actor.ID, group.ID, input.DynamicRule.Expression, time.Now().UTC()); err != nil {
			return writeAdminGroupError(c, err)
		}
	}
	return support.NoStoreJSON(c, http.StatusCreated, toGroupSummaryResponse(group, 0))
}

func HandleUpdateDynamicGroupRule(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input dynamicRuleRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	rule, err := groupusecases.UpdateDynamicGroupRule(c.Request().Context(), dynamicGroupDeps(d), actor.ID, c.Param("group_id"), input.Expression, time.Now().UTC())
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, rule)
}

func HandlePreviewDynamicGroupRule(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input dynamicRulePreviewRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	preview, err := groupusecases.PreviewDynamicGroupRule(c.Request().Context(), dynamicGroupDeps(d), c.Param("group_id"), input.Expression, input.UserIDs)
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"results": preview})
}

func handleSetDynamicGroupRuleEnabled(d Deps, c *echo.Context, enabled bool) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	rule, err := groupusecases.SetDynamicGroupRuleEnabled(c.Request().Context(), dynamicGroupDeps(d), actor.ID, c.Param("group_id"), enabled, time.Now().UTC())
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, rule)
}

func HandleEnableDynamicGroupRule(d Deps, c *echo.Context) error {
	return handleSetDynamicGroupRuleEnabled(d, c, true)
}

func HandleDisableDynamicGroupRule(d Deps, c *echo.Context) error {
	return handleSetDynamicGroupRuleEnabled(d, c, false)
}

func HandleUpdateGroup(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input groupUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	group, err := groupusecases.UpdateGroup(c.Request().Context(), adminGroupDeps(d), groupusecases.UpdateGroupInput{
		ActorUserID: actor.ID, ID: c.Param("group_id"),
		Name: input.Name, Description: input.Description, Roles: input.Roles, Now: time.Now().UTC(),
	})
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	count, err := adminGroupDeps(d).GroupRepo.CountMembers(c.Request().Context(), group.TenantID, group.ID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toGroupSummaryResponse(group, count))
}

func HandleDeleteGroup(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := groupusecases.DeleteGroup(c.Request().Context(), adminGroupDeps(d), actor.ID, c.Param("group_id"), time.Now().UTC()); err != nil {
		return writeAdminGroupError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleAddGroupMember(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := groupusecases.AddMember(c.Request().Context(), adminGroupDeps(d), actor.ID, c.Param("group_id"), c.Param("user_sub"), time.Now().UTC()); err != nil {
		return writeAdminGroupError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleRemoveGroupMember(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := groupusecases.RemoveMember(c.Request().Context(), adminGroupDeps(d), actor.ID, c.Param("group_id"), c.Param("user_sub"), time.Now().UTC()); err != nil {
		return writeAdminGroupError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleListUserGroups(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	view, err := groupusecases.UserGroups(c.Request().Context(), adminGroupDeps(d), c.Param("sub"))
	if err != nil {
		return writeAdminGroupError(c, err)
	}
	groups := make([]groupSummaryResponse, len(view.Groups))
	for i, group := range view.Groups {
		count, err := d.GroupRepo.CountMembers(c.Request().Context(), group.TenantID, group.ID)
		if err != nil {
			return err
		}
		groups[i] = toGroupSummaryResponse(group, count)
	}
	return support.NoStoreJSON(c, http.StatusOK, userGroupsResponse{
		Groups:         groups,
		DirectRoles:    view.DirectRoles,
		GroupRoles:     view.GroupRoles,
		EffectiveRoles: view.EffectiveRoles,
	})
}

func adminGroupDeps(d Deps) groupusecases.AdminGroupDeps {
	return groupusecases.AdminGroupDeps{GroupRepo: d.GroupRepo, UserRepo: d.UserRepo, Emit: d.LegacyEmit(), QuotaRepo: d.QuotaRepo}
}

func dynamicGroupDeps(d Deps) groupusecases.DynamicGroupDeps {
	return groupusecases.DynamicGroupDeps{
		GroupRepo: d.GroupRepo, UserRepo: d.UserRepo, SchemaRepo: d.AttrSchemaRepo, JobRepo: d.JobRepo, Emit: d.LegacyEmit(),
		QuotaRepo: d.QuotaRepo,
	}
}

func toGroupMemberResponses(ctx context.Context, d Deps, members []*groupdomain.GroupMember) []groupMemberResponse {
	out := make([]groupMemberResponse, len(members))
	for i, member := range members {
		username := member.UserID
		if user, err := d.UserRepo.FindBySub(ctx, member.UserID); err == nil && user != nil {
			username = user.PreferredUsername
		}
		out[i] = groupMemberResponse{UserID: member.UserID, PreferredUsername: username, Source: member.Source.Effective(), RuleVersion: member.RuleVersion, CreatedAt: member.CreatedAt}
	}
	return out
}

func toGroupSummaryResponse(group *groupdomain.Group, memberCount int) groupSummaryResponse {
	return groupSummaryResponse{
		ID: group.ID, TenantID: group.TenantID, Name: group.Name, Description: group.Description,
		Roles: slices.Clone(group.Roles), MemberCount: memberCount,
		MembershipType: group.MembershipType.Effective(), CreatedAt: group.CreatedAt, UpdatedAt: group.UpdatedAt,
	}
}

func writeAdminGroupError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, groupusecases.ErrGroupNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "group_not_found", "The group does not exist.")
	case errors.Is(err, idmusecases.ErrUserNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "The user does not exist.")
	case errors.Is(err, groupusecases.ErrGroupNameConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "group_name_conflict", "The group name is already in use.")
	case errors.Is(err, groupusecases.ErrGroupNameEmpty):
		return support.WriteBrowserError(c, http.StatusBadRequest, "group_name_required", "The group name is required.")
	case errors.Is(err, idmusecases.ErrInvalidRole):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_role", "The role is invalid.")
	case errors.Is(err, groupusecases.ErrDynamicMembershipManaged):
		return support.WriteBrowserError(c, http.StatusConflict, "dynamic_membership_managed_by_rule", "Dynamic group membership is managed by its rule.")
	case errors.Is(err, groupusecases.ErrInvalidDynamicGroupRule):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_dynamic_group_rule", err.Error())
	default:
		return err
	}
}

// effectiveRoles は actor の有効ロール (user.roles ∪ 所属 group.roles) を返す (ADR-038)。
// GroupRepo 未配線時は user.roles をそのまま返し、後方互換を保つ。

// withEffectiveRoles は user のコピーに有効ロールを載せて返す (ADR-038)。
// admin actor を解決する各経路 (settings / role policy / key / audit) が
// グループ由来ロールを一貫して評価できるようにする。
