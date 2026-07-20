package usecases

import (
	"fmt"
	"slices"
	"sort"
	"strings"

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
	"AdminApplicationsManage":              {"ListAdminApplications", "GetAdminApplication", "CreateAdminApplication", "UpdateAdminApplication", "DeleteAdminApplication", "AttachProtocolBinding", "DetachProtocolBinding", "UpdateApplicationOidcConfig", "UpdateApplicationWsFedConfig"},
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
	"ManageScimSettings": {
		"ListScimTokens", "CreateScimToken", "RevokeScimToken",
	},
	"BrandingUpdate": {
		"UpdateTenantBranding", "UploadTenantBrandingAsset", "DeleteTenantBrandingAsset",
	},
}

func ListRolePolicies(scl *spec.SCL, actorRoles []string, controlPlane bool) ([]RolePolicy, error) {
	if scl == nil {
		return nil, fmt.Errorf("SCL is required")
	}
	roleDefinitions := []struct {
		name       string
		vocabulary string
	}{
		{name: "admin", vocabulary: "Administrator"},
		{name: "system_admin", vocabulary: "SystemAdministrator"},
	}
	roles := make([]RolePolicy, 0, len(roleDefinitions))
	for _, definition := range roleDefinitions {
		vocabulary, ok := scl.Vocabulary[definition.vocabulary]
		if !ok {
			return nil, fmt.Errorf("vocabulary %s is missing", definition.vocabulary)
		}
		role := RolePolicy{
			Name:        definition.name,
			Description: vocabulary.Definition,
			Aliases:     slices.Clone(vocabulary.Aliases),
		}
		for permissionName := range rolePermissionInterfaces {
			requirements, applies := capabilityRequirements(scl, permissionName, definition.name)
			if !applies {
				continue
			}
			if definition.name == "system_admin" &&
				(!slices.Contains(actorRoles, "system_admin") || !controlPlane) {
				continue
			}
			interfaces, err := rolePolicyInterfaces(scl, permissionName)
			if err != nil {
				return nil, err
			}
			action, ok := spec.ActionNameForCapability(permissionName)
			if !ok {
				return nil, fmt.Errorf("action for permission %s is not mapped", permissionName)
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

func capabilityRequirements(scl *spec.SCL, capabilityName, role string) ([]string, bool) {
	var requirements []string
	for _, interfaceName := range rolePermissionInterfaces[capabilityName] {
		iface, ok := scl.Interfaces[interfaceName]
		if !ok {
			continue
		}
		access, protected := spec.ProtectedInterfaceAccess(iface)
		if !protected {
			continue
		}
		authorization := scl.AuthorizationByContext[scl.InterfaceContexts[interfaceName]]
		for _, policyName := range access.Policies {
			policy, ok := authorization.Policies[policyName]
			if !ok || policy.Effect != "permit" {
				continue
			}
			principal, ok := authorization.Principals[policy.Principal]
			if !ok || !principalAppliesToRole(principal.Matches, role) {
				continue
			}
			requirements = append(requirements, principal.Matches...)
			if policy.When != "" {
				requirements = append(requirements, policy.When)
			}
		}
	}
	if len(requirements) == 0 {
		return nil, false
	}
	sort.Strings(requirements)
	return slices.Compact(requirements), true
}

func principalAppliesToRole(requirements []string, role string) bool {
	for _, requirement := range requirements {
		switch role {
		case "admin":
			clean := strings.ReplaceAll(requirement, "system_admin", "")
			if strings.Contains(clean, "admin") && strings.Contains(requirement, "principal.roles") {
				return true
			}
		case "system_admin":
			if strings.Contains(requirement, "system_admin") && strings.Contains(requirement, "principal.roles") {
				return true
			}
		}
	}
	return false
}

func rolePolicyInterfaces(scl *spec.SCL, permissionName string) ([]RoleInterface, error) {
	names, ok := rolePermissionInterfaces[permissionName]
	if !ok {
		return nil, fmt.Errorf("interfaces for permission %s are not mapped", permissionName)
	}
	interfaces := make([]RoleInterface, 0, len(names))
	for _, name := range names {
		iface, ok := scl.Interfaces[name]
		if !ok {
			return nil, fmt.Errorf("interface %s for permission %s is missing", name, permissionName)
		}
		binding, ok := scl.HTTPBinding(iface)
		if !ok {
			return nil, fmt.Errorf("HTTP binding for interface %s is missing", name)
		}
		interfaces = append(interfaces, RoleInterface{
			Name:   name,
			Method: binding.String("method"),
			Path:   binding.String("path"),
		})
	}
	return interfaces, nil
}
