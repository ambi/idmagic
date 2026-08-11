package usecases

import (
	"fmt"
	"slices"
	"sort"

	"github.com/ambi/idmagic/backend/shared/spec"
)

type RolePolicy struct {
	Name        string
	Description string
	Aliases     []string
	Permissions []RolePermission
}

type RolePermission struct {
	Name         string
	Action       string
	Description  string
	Requirements []string
	Interfaces   []RoleInterface
}

type RoleInterface struct {
	Name   string
	Method string
	Path   string
}

var rolePermissionInterfaces = map[string][]string{
	"AdminUserRead":                        {"ListAdminUsers", "GetAdminUser"},
	"AdminUserCreate":                      {"CreateAdminUser"},
	"AdminUserImport":                      {"ImportAdminUsers", "GetAdminUserImport"},
	"AdminUserUpdate":                      {"UpdateAdminUser", "DisableAdminUser", "EnableAdminUser"},
	"AdminUserDelete":                      {"DeleteAdminUser"},
	"AdminUserRestore":                     {"RestoreAdminUser"},
	"AdminUserPurge":                       {"DeleteAdminUser"},
	"AdminOAuth2ClientsManage":             {"ListAdminOAuth2Clients", "GetAdminOAuth2Client", "CreateAdminOAuth2Client", "UpdateAdminOAuth2Client", "DeleteAdminOAuth2Client"},
	"AdminConsentsManage":                  {"ListAdminConsents", "GetAdminConsent", "RevokeAdminConsent"},
	"AdminTenantsManage":                   {"ListTenants", "GetTenant", "CreateTenant", "UpdateTenant", "DisableTenant", "EnableTenant"},
	"AdminSettingsRead":                    {"GetAdminSettings"},
	"AdminSettingsUpdate":                  {"UpdateAdminSettings"},
	"AdminAuditEventsRead":                 {"ListAdminAuditEvents", "ExportAdminAuditEvents", "GetAdminAuditEvent"},
	"AdminKeysRead":                        {"ListAdminKeys", "GetAdminKey"},
	"TenantKeysRotate":                     {"RotateTenantSigningKey"},
	"TenantKeysDisable":                    {"DisableTenantKey"},
	"SystemKeyHealthRead":                  {"ListTenantKeyHealth"},
	"AdminGroupsRead":                      {"ListGroups", "GetGroup", "ListUserGroups"},
	"AdminGroupsWrite":                     {"CreateGroup", "UpdateGroup", "DeleteGroup", "AddGroupMember", "RemoveGroupMember"},
	"AdminAgentsManage":                    {"ListAgents", "GetAgent", "RegisterAgent", "UpdateAgent", "DisableAgent", "EnableAgent", "KillAgent", "DeleteAgent", "BindAgentCredential", "UnbindAgentCredential"},
	"AdminAuthorizationDetailTypesManage":  {"ListAuthorizationDetailTypes", "GetAuthorizationDetailType", "CreateAuthorizationDetailType", "UpdateAuthorizationDetailType", "DeleteAuthorizationDetailType"},
	"AdminApplicationsManage":              {"ListAdminApplications", "GetAdminApplication", "CreateAdminApplication", "UpdateAdminApplication", "DeleteAdminApplication", "UpdateApplicationOidcConfig", "UpdateApplicationWsFedConfig", "UpdateApplicationSamlConfig"},
	"AdminApplicationAssignmentsManage":    {"ListApplicationAssignments", "AssignApplication", "UnassignApplication"},
	"AdminApplicationPoliciesManage":       {"GetAppSignInPolicy", "UpdateAppSignInPolicy"},
	"AdminTenantDefaultSignInPolicyManage": {"GetTenantDefaultSignInPolicy", "UpdateTenantDefaultSignInPolicy"},
	"AdminApplicationCategoriesManage":     {"ListApplicationCategories", "CreateApplicationCategory", "UpdateApplicationCategory", "DeleteApplicationCategory", "SetApplicationCategories"},
	"AdminFederationTrustsManage":          {"RegisterSamlServiceProvider", "ListSamlServiceProviders", "DeleteSamlServiceProvider", "RegisterWsFedRelyingParty", "ListWsFedRelyingParties", "DeleteWsFedRelyingParty"},
	"ScimProvision": {
		"GetScimServiceProviderConfig", "GetScimResourceTypes", "GetScimSchemas",
		"CreateScimUser", "GetScimUser", "PatchScimUser", "UpdateScimUser", "DeleteScimUser",
		"CreateScimGroup", "GetScimGroup", "PatchScimGroup", "UpdateScimGroup", "DeleteScimGroup",
	},
	"ManageScimSettings": {},
	"BrandingUpdate": {
		"UpdateTenantBranding", "UploadTenantBrandingAsset", "DeleteTenantBrandingAsset",
	},
}

func ListRolePolicies(contract *spec.RuntimeContract, actorRoles []string, controlPlane bool) ([]RolePolicy, error) {
	if contract == nil {
		return nil, fmt.Errorf("runtime contract is required")
	}
	roleDefinitions := []struct {
		name        string
		description string
		aliases     []string
	}{
		{
			name:        "admin",
			description: "User.roles に admin を持ち、所属テナント内の管理 API を許可された認証済みユーザー。",
			aliases:     []string{"admin", "管理者", "TenantAdmin"},
		},
		{
			name:        "system_admin",
			description: "User.roles に system_admin を持ち、テナント境界を越える管理操作を許可された認証済みユーザー。",
			aliases:     []string{"system_admin", "システム管理者"},
		},
	}
	roles := make([]RolePolicy, 0, len(roleDefinitions))
	for _, definition := range roleDefinitions {
		role := RolePolicy{
			Name:        definition.name,
			Description: definition.description,
			Aliases:     slices.Clone(definition.aliases),
		}
		for permissionName := range rolePermissionInterfaces {
			action, ok := spec.ActionNameForCapability(permissionName)
			if !ok {
				return nil, fmt.Errorf("action for permission %s is not mapped", permissionName)
			}
			requirements, applies := capabilityRequirements(action, definition.name)
			if !applies {
				continue
			}
			if definition.name == "system_admin" &&
				(!slices.Contains(actorRoles, "system_admin") || !controlPlane) {
				continue
			}
			interfaces, err := rolePolicyInterfaces(contract, permissionName)
			if err != nil {
				return nil, err
			}
			role.Permissions = append(role.Permissions, RolePermission{
				Name:         permissionName,
				Action:       action,
				Requirements: requirements,
				Interfaces:   interfaces,
			})
		}
		sort.Slice(role.Permissions, func(i, j int) bool {
			return role.Permissions[i].Name < role.Permissions[j].Name
		})
		roles = append(roles, role)
	}
	return roles, nil
}

func capabilityRequirements(action, role string) ([]string, bool) {
	requirements, ok := spec.RulesForAction(action)
	if !ok {
		return nil, false
	}
	applies := false
	for _, requirement := range requirements {
		switch role {
		case "admin":
			applies = applies || requirement == "actor_is_admin" || requirement == "actor_is_admin_or_system_admin"
		case "system_admin":
			applies = applies || requirement == "actor_is_system_admin" || requirement == "actor_is_admin_or_system_admin"
		}
	}
	if !applies {
		return nil, false
	}
	sort.Strings(requirements)
	return slices.Compact(requirements), true
}

func rolePolicyInterfaces(contract *spec.RuntimeContract, permissionName string) ([]RoleInterface, error) {
	names, ok := rolePermissionInterfaces[permissionName]
	if !ok {
		return nil, fmt.Errorf("interfaces for permission %s are not mapped", permissionName)
	}
	interfaces := make([]RoleInterface, 0, len(names))
	for _, name := range names {
		operation, ok := contract.Operation(name)
		if !ok {
			return nil, fmt.Errorf("interface %s for permission %s is missing", name, permissionName)
		}
		interfaces = append(interfaces, RoleInterface{
			Name:   name,
			Method: operation.Method,
			Path:   operation.Path,
		})
	}
	return interfaces, nil
}
