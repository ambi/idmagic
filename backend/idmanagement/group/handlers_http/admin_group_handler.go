package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sync"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type groupCreateRequest struct {
	Name           string                               `json:"name"`
	Description    *string                              `json:"description"`
	Email          *string                              `json:"email"`
	Attributes     map[string]userdomain.AttributeValue `json:"attributes"`
	Roles          []string                             `json:"roles"`
	MembershipType groupdomain.GroupMembershipType      `json:"membership_type"`
	DynamicRule    *dynamicRuleRequest                  `json:"dynamic_rule"`
}

type dynamicRuleRequest struct {
	Expression string `json:"expression"`
}

type dynamicRulePreviewRequest struct {
	Expression string   `json:"expression"`
	UserIDs    []string `json:"user_ids"`
}

type groupUpdateRequest struct {
	Name        *string                               `json:"name"`
	Description *string                               `json:"description"`
	Email       *string                               `json:"email"`
	Attributes  *map[string]userdomain.AttributeValue `json:"attributes"`
	Roles       *[]string                             `json:"roles"`
}

type groupSummaryResponse struct {
	ID             string                               `json:"id"`
	TenantID       string                               `json:"tenant_id"`
	Name           string                               `json:"name"`
	Description    *string                              `json:"description,omitempty"`
	Email          *string                              `json:"email,omitempty"`
	Attributes     map[string]userdomain.AttributeValue `json:"attributes"`
	Roles          []string                             `json:"roles"`
	MemberCount    int                                  `json:"member_count"`
	CreatedAt      time.Time                            `json:"created_at"`
	UpdatedAt      time.Time                            `json:"updated_at"`
	ScimSource     *string                              `json:"scim_source,omitempty"`
	MembershipType groupdomain.GroupMembershipType      `json:"membership_type"`
	DynamicRule    *groupdomain.DynamicGroupRule        `json:"dynamic_rule,omitempty"`
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

const (
	listGroupsQuery        = "ListGroups"
	listGroupsDefaultLimit = 50
	listGroupsMaxLimit     = 200
)

func HandleListGroups(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	tenantID := support.RequestTenantID(c)
	page, err := support.ParsePageRequest(c, d.PaginationCodec, tenantID, listGroupsQuery, listGroupsDefaultLimit, listGroupsMaxLimit)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	ctx := c.Request().Context()
	var views []groupusecases.GroupView
	var pageErr, countErr error
	var totalItems int64
	var wg sync.WaitGroup
	wg.Go(func() {
		if page.Direction == support.PageBackward {
			views, pageErr = groupusecases.ListGroupsBefore(ctx, adminGroupDeps(d), page.AfterPrimary, page.AfterID, page.Limit+1)
		} else {
			views, pageErr = groupusecases.ListGroups(ctx, adminGroupDeps(d), page.AfterPrimary, page.AfterID, page.Limit+1)
		}
	})
	wg.Go(func() { totalItems, countErr = d.GroupRepo.Count(ctx, tenantID) })
	wg.Wait()
	if pageErr != nil {
		return pageErr
	}
	if countErr != nil {
		return countErr
	}
	views, hasPrevious, hasNext := support.TrimPage(views, page)
	if page.Anchor == support.PageAnchorEnd {
		views = support.TrimEndPage(views, totalItems, page.Limit)
	}
	metadata := support.CalculatePaginationMetadata(totalItems, page)
	support.SetPaginationHeaders(c, metadata)
	groups := make([]groupSummaryResponse, len(views))
	for i, view := range views {
		groups[i] = toGroupSummaryResponse(view.Group, view.MemberCount)
	}
	var firstPrimary, firstID, lastPrimary, lastID string
	if len(views) > 0 {
		first := views[0]
		last := views[len(views)-1]
		firstPrimary, firstID = first.Group.Name, first.Group.ID
		lastPrimary, lastID = last.Group.Name, last.Group.ID
	}
	if err := support.SetPaginationLinks(c, d.PaginationCodec, d.Issuer, tenantID, listGroupsQuery, page,
		firstPrimary, firstID, lastPrimary, lastID, hasPrevious, hasNext, metadata.TotalPages); err != nil {
		return err
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
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
		ActorUserID: actor.ID, Name: input.Name, Description: input.Description,
		Email: input.Email, Attributes: input.Attributes,
		Roles: input.Roles, MembershipType: input.MembershipType, Now: time.Now().UTC(),
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
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
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	group, err := groupusecases.UpdateGroup(c.Request().Context(), adminGroupDeps(d), groupusecases.UpdateGroupInput{
		ActorUserID: actor.ID, ID: c.Param("group_id"),
		Name: input.Name, Description: input.Description,
		Email: input.Email, Attributes: input.Attributes,
		Roles: input.Roles, Now: time.Now().UTC(),
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
	return groupusecases.AdminGroupDeps{
		GroupRepo: d.GroupRepo, UserRepo: d.UserRepo, Emit: d.LegacyEmit(), QuotaRepo: d.QuotaRepo,
		GroupAttrSchemaRepo: d.GroupAttrSchemaRepo,
	}
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
	attributes := group.Attributes
	if attributes == nil {
		attributes = map[string]userdomain.AttributeValue{}
	}
	return groupSummaryResponse{
		ID: group.ID, TenantID: group.TenantID, Name: group.Name, Description: group.Description,
		Email: group.Email, Attributes: attributes,
		Roles: slices.Clone(group.Roles), MemberCount: memberCount,
		MembershipType: group.MembershipType.Effective(), CreatedAt: group.CreatedAt, UpdatedAt: group.UpdatedAt,
	}
}

func writeAdminGroupError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, groupusecases.ErrGroupNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "group_not_found", "The group does not exist.")
	case errors.Is(err, idmusecases.ErrUserNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "user_not_found", "The user does not exist.")
	case errors.Is(err, groupusecases.ErrGroupNameConflict):
		return support.WriteProblem(c, http.StatusConflict, "group_name_conflict", "The group name is already in use.")
	case errors.Is(err, groupusecases.ErrGroupNameEmpty):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "group_name_required", "The group name is required.")
	case errors.Is(err, idmusecases.ErrInvalidRole):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_role", "The role is invalid.")
	case errors.Is(err, groupusecases.ErrDynamicMembershipManaged):
		return support.WriteProblem(c, http.StatusConflict, "dynamic_membership_managed_by_rule", "Dynamic group membership is managed by its rule.")
	case errors.Is(err, groupusecases.ErrInvalidDynamicGroupRule):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_dynamic_group_rule", err.Error())
	case errors.Is(err, groupusecases.ErrInvalidEmail):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_email", "The email address is not valid.")
	case errors.Is(err, groupusecases.ErrInvalidAttribute):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "invalid_attribute", "The attribute does not conform to the schema.")
	default:
		return err
	}
}

// effectiveRoles は actor の有効ロール (user.roles ∪ 所属 group.roles) を返す。
// GroupRepo 未配線時は user.roles をそのまま返し、後方互換を保つ。

// withEffectiveRoles は user のコピーに有効ロールを載せて返す。
// admin actor を解決する各経路 (settings / role policy / key / audit) が
// グループ由来ロールを一貫して評価できるようにする。
