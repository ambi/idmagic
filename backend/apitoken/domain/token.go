package domain

import (
	"errors"
	"slices"
	"sort"
	"time"
)

const BuiltinClientID = "idmagic-api-token"

var (
	ErrInvalidToken = errors.New("invalid API token")
	ErrInvalidScope = errors.New("invalid API token scope")
)

type Scope string

const (
	ScopeUsersRead                     Scope = "users:read"
	ScopeUsersWrite                    Scope = "users:write"
	ScopeGroupsRead                    Scope = "groups:read"
	ScopeGroupsWrite                   Scope = "groups:write"
	ScopeAgentsRead                    Scope = "agents:read"
	ScopeAgentsWrite                   Scope = "agents:write"
	ScopeSessionsRead                  Scope = "sessions:read"
	ScopeSessionsWrite                 Scope = "sessions:write"
	ScopeConsentsRead                  Scope = "consents:read"
	ScopeConsentsWrite                 Scope = "consents:write"
	ScopeLifecycleWorkflowsRead        Scope = "lifecycle-workflows:read"
	ScopeLifecycleWorkflowsWrite       Scope = "lifecycle-workflows:write"
	ScopeTenantsRead                   Scope = "tenants:read"
	ScopeTenantsWrite                  Scope = "tenants:write"
	ScopeSettingsRead                  Scope = "settings:read"
	ScopeSettingsWrite                 Scope = "settings:write"
	ScopeSigningKeysRead               Scope = "signing-keys:read"
	ScopeSigningKeysWrite              Scope = "signing-keys:write"
	ScopeAuditRead                     Scope = "audit:read"
	ScopeApplicationsRead              Scope = "applications:read"
	ScopeApplicationsWrite             Scope = "applications:write"
	ScopeOAuthClientsRead              Scope = "oauth-clients:read"
	ScopeOAuthClientsWrite             Scope = "oauth-clients:write"
	ScopeAuthorizationDetailTypesRead  Scope = "authorization-detail-types:read"
	ScopeAuthorizationDetailTypesWrite Scope = "authorization-detail-types:write"
	ScopeMcpResourceServersRead        Scope = "mcp-resource-servers:read"
	ScopeMcpResourceServersWrite       Scope = "mcp-resource-servers:write"
	ScopeSamlRead                      Scope = "saml:read"
	ScopeSamlWrite                     Scope = "saml:write"
	ScopeWsFedRead                     Scope = "wsfed:read"
	ScopeWsFedWrite                    Scope = "wsfed:write"
	ScopeProvisioningRead              Scope = "provisioning:read"
	ScopeProvisioningWrite             Scope = "provisioning:write"
	ScopeScimUsersRead                 Scope = "scim:users:read"
	ScopeScimUsersWrite                Scope = "scim:users:write"
	ScopeScimGroupsRead                Scope = "scim:groups:read"
	ScopeScimGroupsWrite               Scope = "scim:groups:write"
	ScopeAccountRead                   Scope = "account:read"
	ScopeAccountWrite                  Scope = "account:write"
	ScopeAccountMFAWrite               Scope = "account:mfa:write"
	ScopeAccountSessionsWrite          Scope = "account:sessions:write"
	ScopeAccountConsentsWrite          Scope = "account:consents:write"
	ScopeAccountPasswordWrite          Scope = "account:password:write"
)

var validScopes = map[Scope]struct{}{
	ScopeUsersRead: {}, ScopeUsersWrite: {}, ScopeGroupsRead: {}, ScopeGroupsWrite: {},
	ScopeAgentsRead: {}, ScopeAgentsWrite: {}, ScopeSessionsRead: {}, ScopeSessionsWrite: {},
	ScopeConsentsRead: {}, ScopeConsentsWrite: {}, ScopeLifecycleWorkflowsRead: {},
	ScopeLifecycleWorkflowsWrite: {}, ScopeTenantsRead: {}, ScopeTenantsWrite: {},
	ScopeSettingsRead: {}, ScopeSettingsWrite: {}, ScopeSigningKeysRead: {},
	ScopeSigningKeysWrite: {}, ScopeAuditRead: {}, ScopeApplicationsRead: {},
	ScopeApplicationsWrite: {}, ScopeOAuthClientsRead: {}, ScopeOAuthClientsWrite: {},
	ScopeAuthorizationDetailTypesRead: {}, ScopeAuthorizationDetailTypesWrite: {},
	ScopeMcpResourceServersRead: {}, ScopeMcpResourceServersWrite: {}, ScopeSamlRead: {},
	ScopeSamlWrite: {}, ScopeWsFedRead: {}, ScopeWsFedWrite: {}, ScopeProvisioningRead: {},
	ScopeProvisioningWrite: {}, ScopeScimUsersRead: {},
	ScopeScimUsersWrite: {}, ScopeScimGroupsRead: {}, ScopeScimGroupsWrite: {},
	ScopeAccountRead: {}, ScopeAccountWrite: {}, ScopeAccountMFAWrite: {},
	ScopeAccountSessionsWrite: {}, ScopeAccountConsentsWrite: {}, ScopeAccountPasswordWrite: {},
}

type Scopes []Scope

func ParseScopes(values []string) (Scopes, error) {
	result := make(Scopes, 0, len(values))
	seen := make(map[Scope]struct{}, len(values))
	for _, value := range values {
		scope := Scope(value)
		if _, ok := validScopes[scope]; !ok {
			return nil, ErrInvalidScope
		}
		if _, duplicate := seen[scope]; duplicate {
			continue
		}
		seen[scope] = struct{}{}
		result = append(result, scope)
	}
	return result, nil
}

func (s Scopes) Has(scope Scope) bool {
	return slices.Contains(s, scope)
}

func (s Scopes) HasAny(scopes ...Scope) bool {
	return slices.ContainsFunc(scopes, s.Has)
}

func (s Scopes) Strings() []string {
	result := make([]string, len(s))
	for i, scope := range s {
		result[i] = string(scope)
	}
	return result
}

func AllScopes() []string {
	result := make([]string, 0, len(validScopes))
	for scope := range validScopes {
		result = append(result, string(scope))
	}
	sort.Strings(result)
	return result
}

type ApiToken struct {
	ID          string
	TenantID    string
	UserID      string
	JTI         string
	ClientID    string
	Scopes      Scopes
	Audience    string
	DPoPJKT     string
	Description string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
}

type Metadata struct {
	ID          string
	JTI         string
	UserID      string
	ClientID    string
	Description string
	Scopes      Scopes
	Audience    string
	DPoPJKT     string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	RevokedAt   *time.Time
}

func (t *ApiToken) Metadata() Metadata {
	return Metadata{
		ID: t.ID, JTI: t.JTI, UserID: t.UserID, ClientID: t.ClientID,
		Description: t.Description, Scopes: append(Scopes(nil), t.Scopes...),
		Audience: t.Audience, DPoPJKT: t.DPoPJKT, CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt, RevokedAt: t.RevokedAt,
	}
}

type Principal struct {
	TenantID  string
	UserID    string
	ClientID  string
	Scopes    Scopes
	Audience  string
	TokenID   string
	IssuedAt  time.Time
	ExpiresAt *time.Time
	DPoPJKT   string
}
