package handlers_http

import (
	"net/http"
	"slices"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// roleDescriptionCopy / permissionDescriptionCopy は利用者向けの平易な説明。
// SCL の normative 定義は設計者向け語彙 (User.roles / SystemAdministrator /
// tombstone 等) を含むため、管理 UI には用語集を引いた user-facing コピーを返す。
// 未登録の名前は raw の説明文をそのままフォールバックする。
var roleDescriptionCopy = map[string]string{
	"admin":        "An administrator role that can manage users, applications, groups, and settings within its own tenant. Cannot perform cross-tenant operations.",
	"system_admin": "A system-wide administrator role. Can perform cross-tenant operations such as creating or disabling tenants.",
}

var permissionDescriptionCopy = map[string]string{
	"AdminUserRead":                 "View the user list and user details.",
	"AdminUserCreate":               "Create a new user.",
	"AdminUserImport":               "Validate and bulk-register users via CSV.",
	"AdminUserUpdate":               "Update a user's profile, roles, and enabled status.",
	"AdminUserDelete":               "Schedule a user for deletion. Can be restored within the recovery window.",
	"AdminUserRestore":              "Restore a user scheduled for deletion.",
	"AdminUserPurge":                "Permanently delete a user and anonymize their personal information. Cannot be undone.",
	"AdminOAuth2ClientsManage":      "Register, update, and delete OAuth2/OIDC clients.",
	"AdminConsentsManage":           "View and revoke consents users have granted to OAuth2/OIDC clients.",
	"AdminTenantsManage":            "Create, update, disable, and enable tenants.",
	"AdminSettingsRead":             "View tenant settings.",
	"AdminSettingsUpdate":           "Update tenant settings.",
	"AdminAuditEventsRead":          "View audit logs.",
	"AdminKeysRead":                 "View signing keys.",
	"TenantKeysRotate":              "Rotate the signing key for your own tenant.",
	"TenantKeysDisable":             "Emergency-disable the signing key for your own tenant.",
	"SystemKeyHealthRead":           "View signing key health across tenants.",
	"AdminGroupsRead":               "View the group list and group details.",
	"AdminGroupsWrite":              "Create, update, and delete groups, and manage their members.",
	"AdminAgentsManage":             "Register, update, disable, kill, and delete AI agents (non-human identities), and bind their credentials.",
	"AdminAuthorizationModelManage": "Publish the fine-grained authorization model, write relation tuples, and run access checks for your own tenant.",
}

func roleDescription(name, raw string) string {
	if text, ok := roleDescriptionCopy[name]; ok {
		return text
	}
	return raw
}

func permissionDescription(name, raw string) string {
	if text, ok := permissionDescriptionCopy[name]; ok {
		return text
	}
	return raw
}

type AdminRolePolicyResponse struct {
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Aliases     []string                      `json:"aliases"`
	Permissions []adminRolePermissionResponse `json:"permissions"`
}

type adminRolePermissionResponse struct {
	Name        string                       `json:"name"`
	Action      string                       `json:"action"`
	Description string                       `json:"description"`
	Interfaces  []adminRoleInterfaceResponse `json:"interfaces"`
}

type adminRoleInterfaceResponse struct {
	Name   string `json:"name"`
	Method string `json:"method"`
	Path   string `json:"path"`
}

func (d Deps) handleListAdminRolePolicies(c *echo.Context) error {
	actor, err := d.ResolveAdminActor(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if !slices.Contains(actor.Roles, "admin") && !slices.Contains(actor.Roles, "system_admin") {
		return d.WriteAdminAccessError(c, support.ErrAdminAccessDenied)
	}
	roles, err := tokenusecases.ListRolePolicies(
		d.Contract,
		actor.Roles,
		support.RequestTenantID(c) == tenancydomain.DefaultTenantID && actor.TenantID == tenancydomain.DefaultTenantID,
	)
	if err != nil {
		return err
	}
	response := make([]AdminRolePolicyResponse, len(roles))
	for i, role := range roles {
		response[i] = toAdminRolePolicyResponse(role)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"roles": response})
}

func toAdminRolePolicyResponse(role tokenusecases.RolePolicy) AdminRolePolicyResponse {
	permissions := make([]adminRolePermissionResponse, len(role.Permissions))
	for i, permission := range role.Permissions {
		interfaces := make([]adminRoleInterfaceResponse, len(permission.Interfaces))
		for j, iface := range permission.Interfaces {
			interfaces[j] = adminRoleInterfaceResponse(iface)
		}
		permissions[i] = adminRolePermissionResponse{
			Name: permission.Name, Action: permission.Action,
			Description: permissionDescription(permission.Name, permission.Description), Interfaces: interfaces,
		}
	}
	return AdminRolePolicyResponse{
		Name: role.Name, Description: roleDescription(role.Name, role.Description), Aliases: slices.Clone(role.Aliases),
		Permissions: permissions,
	}
}
