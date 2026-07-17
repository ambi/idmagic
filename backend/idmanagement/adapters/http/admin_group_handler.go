package http

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type groupCreateRequest struct {
	Name           string                        `json:"name"`
	Description    *string                       `json:"description"`
	Roles          []string                      `json:"roles"`
	MembershipType idmdomain.GroupMembershipType `json:"membership_type"`
	DynamicRule    *dynamicRuleRequest           `json:"dynamic_rule"`
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
	ID             string                        `json:"id"`
	TenantID       string                        `json:"tenant_id"`
	Name           string                        `json:"name"`
	Description    *string                       `json:"description,omitempty"`
	Roles          []string                      `json:"roles"`
	MemberCount    int                           `json:"member_count"`
	CreatedAt      time.Time                     `json:"created_at"`
	UpdatedAt      time.Time                     `json:"updated_at"`
	ScimSource     *string                       `json:"scim_source,omitempty"`
	MembershipType idmdomain.GroupMembershipType `json:"membership_type"`
	DynamicRule    *idmdomain.DynamicGroupRule   `json:"dynamic_rule,omitempty"`
}

type groupMemberResponse struct {
	UserID            string                          `json:"user_id"`
	PreferredUsername string                          `json:"preferred_username"`
	Source            idmdomain.GroupMembershipSource `json:"source"`
	RuleVersion       *int64                          `json:"rule_version,omitempty"`
	CreatedAt         time.Time                       `json:"created_at"`
}

type userGroupsResponse struct {
	Groups         []groupSummaryResponse `json:"groups"`
	DirectRoles    []string               `json:"direct_roles"`
	GroupRoles     []string               `json:"group_roles"`
	EffectiveRoles []string               `json:"effective_roles"`
}

func (d Deps) handleListGroups(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	views, err := idmusecases.ListGroups(c.Request().Context(), d.adminGroupDeps())
	if err != nil {
		return err
	}
	groups := make([]groupSummaryResponse, len(views))
	for i, view := range views {
		groups[i] = toGroupSummaryResponse(view.Group, view.MemberCount)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"groups": groups})
}

func (d Deps) handleGetGroup(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	group, members, err := idmusecases.GetGroup(c.Request().Context(), d.adminGroupDeps(), c.Param("group_id"))
	if err != nil {
		return d.writeAdminGroupError(c, err)
	}
	res := toGroupSummaryResponse(group, len(members))
	if group.MembershipType.Effective() == idmdomain.GroupMembershipDynamic {
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
		"members": d.toGroupMemberResponses(c.Request().Context(), members),
	})
}

func (d Deps) handleCreateGroup(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input groupCreateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if input.MembershipType.Effective() == idmdomain.GroupMembershipDynamic && input.DynamicRule != nil {
		defs := idmdomain.BuiltinUserAttributeDefs()
		if d.AttrSchemaRepo != nil {
			schema, schemaErr := d.AttrSchemaRepo.FindByTenant(c.Request().Context(), actor.TenantID)
			if schemaErr != nil {
				return schemaErr
			}
			if schema != nil {
				defs = schema.EffectiveDefs()
			}
		}
		if _, compileErr := idmdomain.CompileDynamicGroupRule(input.DynamicRule.Expression, defs); compileErr != nil {
			return d.writeAdminGroupError(c, errors.Join(idmusecases.ErrInvalidDynamicGroupRule, compileErr))
		}
	}
	group, err := idmusecases.CreateGroup(c.Request().Context(), d.adminGroupDeps(), idmusecases.CreateGroupInput{
		ActorUserID: actor.ID, Name: input.Name, Description: input.Description, Roles: input.Roles, MembershipType: input.MembershipType, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeAdminGroupError(c, err)
	}
	if input.DynamicRule != nil && group.MembershipType.Effective() == idmdomain.GroupMembershipDynamic {
		if _, err := idmusecases.UpdateDynamicGroupRule(c.Request().Context(), d.dynamicGroupDeps(), actor.ID, group.ID, input.DynamicRule.Expression, time.Now().UTC()); err != nil {
			return d.writeAdminGroupError(c, err)
		}
	}
	return support.NoStoreJSON(c, http.StatusCreated, toGroupSummaryResponse(group, 0))
}

func (d Deps) handleUpdateDynamicGroupRule(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input dynamicRuleRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	rule, err := idmusecases.UpdateDynamicGroupRule(c.Request().Context(), d.dynamicGroupDeps(), actor.ID, c.Param("group_id"), input.Expression, time.Now().UTC())
	if err != nil {
		return d.writeAdminGroupError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, rule)
}

func (d Deps) handlePreviewDynamicGroupRule(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input dynamicRulePreviewRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	preview, err := idmusecases.PreviewDynamicGroupRule(c.Request().Context(), d.dynamicGroupDeps(), c.Param("group_id"), input.Expression, input.UserIDs)
	if err != nil {
		return d.writeAdminGroupError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"results": preview})
}

func (d Deps) handleSetDynamicGroupRuleEnabled(c *echo.Context, enabled bool) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	rule, err := idmusecases.SetDynamicGroupRuleEnabled(c.Request().Context(), d.dynamicGroupDeps(), actor.ID, c.Param("group_id"), enabled, time.Now().UTC())
	if err != nil {
		return d.writeAdminGroupError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, rule)
}

func (d Deps) handleEnableDynamicGroupRule(c *echo.Context) error {
	return d.handleSetDynamicGroupRuleEnabled(c, true)
}

func (d Deps) handleDisableDynamicGroupRule(c *echo.Context) error {
	return d.handleSetDynamicGroupRuleEnabled(c, false)
}

func (d Deps) handleUpdateGroup(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input groupUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	group, err := idmusecases.UpdateGroup(c.Request().Context(), d.adminGroupDeps(), idmusecases.UpdateGroupInput{
		ActorUserID: actor.ID, ID: c.Param("group_id"),
		Name: input.Name, Description: input.Description, Roles: input.Roles, Now: time.Now().UTC(),
	})
	if err != nil {
		return d.writeAdminGroupError(c, err)
	}
	count, err := d.adminGroupDeps().GroupRepo.CountMembers(c.Request().Context(), group.TenantID, group.ID)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toGroupSummaryResponse(group, count))
}

func (d Deps) handleDeleteGroup(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := idmusecases.DeleteGroup(c.Request().Context(), d.adminGroupDeps(), actor.ID, c.Param("group_id"), time.Now().UTC()); err != nil {
		return d.writeAdminGroupError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleAddGroupMember(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := idmusecases.AddMember(c.Request().Context(), d.adminGroupDeps(), actor.ID, c.Param("group_id"), c.Param("user_sub"), time.Now().UTC()); err != nil {
		return d.writeAdminGroupError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleRemoveGroupMember(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := idmusecases.RemoveMember(c.Request().Context(), d.adminGroupDeps(), actor.ID, c.Param("group_id"), c.Param("user_sub"), time.Now().UTC()); err != nil {
		return d.writeAdminGroupError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleListUserGroups(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	view, err := idmusecases.UserGroups(c.Request().Context(), d.adminGroupDeps(), c.Param("sub"))
	if err != nil {
		return d.writeAdminGroupError(c, err)
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

func (d Deps) adminGroupDeps() idmusecases.AdminGroupDeps {
	return idmusecases.AdminGroupDeps{GroupRepo: d.GroupRepo, UserRepo: d.UserRepo, Emit: d.legacyEmit()}
}

func (d Deps) dynamicGroupDeps() idmusecases.DynamicGroupDeps {
	return idmusecases.DynamicGroupDeps{GroupRepo: d.GroupRepo, UserRepo: d.UserRepo, SchemaRepo: d.AttrSchemaRepo, JobRepo: d.JobRepo, Emit: d.legacyEmit()}
}

func (d Deps) toGroupMemberResponses(ctx context.Context, members []*idmdomain.GroupMember) []groupMemberResponse {
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

func toGroupSummaryResponse(group *idmdomain.Group, memberCount int) groupSummaryResponse {
	return groupSummaryResponse{
		ID: group.ID, TenantID: group.TenantID, Name: group.Name, Description: group.Description,
		Roles: slices.Clone(group.Roles), MemberCount: memberCount,
		MembershipType: group.MembershipType.Effective(), CreatedAt: group.CreatedAt, UpdatedAt: group.UpdatedAt,
	}
}

func (d Deps) writeAdminGroupError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, idmusecases.ErrGroupNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "group_not_found", "グループが存在しません")
	case errors.Is(err, idmusecases.ErrUserNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "ユーザーが存在しません")
	case errors.Is(err, idmusecases.ErrGroupNameConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "group_name_conflict", "グループ名は既に使用されています")
	case errors.Is(err, idmusecases.ErrGroupNameEmpty):
		return support.WriteBrowserError(c, http.StatusBadRequest, "group_name_required", "グループ名は必須です")
	case errors.Is(err, idmusecases.ErrInvalidRole):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_role", "roleが不正です")
	case errors.Is(err, idmusecases.ErrDynamicMembershipManaged):
		return support.WriteBrowserError(c, http.StatusConflict, "dynamic_membership_managed_by_rule", "動的グループの所属はルールで管理されます")
	case errors.Is(err, idmusecases.ErrInvalidDynamicGroupRule):
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
