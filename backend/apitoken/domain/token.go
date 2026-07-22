package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"
)

const TokenPrefix = "idmagic_pat_"

var (
	ErrInvalidToken = errors.New("invalid API token")
	ErrInvalidScope = errors.New("invalid API token scope")
)

type Scope string

const (
	ScopeUsersRead               Scope = "users:read"
	ScopeUsersWrite              Scope = "users:write"
	ScopeGroupsRead              Scope = "groups:read"
	ScopeGroupsWrite             Scope = "groups:write"
	ScopeAgentsRead              Scope = "agents:read"
	ScopeAgentsWrite             Scope = "agents:write"
	ScopeSessionsRead            Scope = "sessions:read"
	ScopeSessionsWrite           Scope = "sessions:write"
	ScopeConsentsRead            Scope = "consents:read"
	ScopeConsentsWrite           Scope = "consents:write"
	ScopeLifecycleWorkflowsRead  Scope = "lifecycle-workflows:read"
	ScopeLifecycleWorkflowsWrite Scope = "lifecycle-workflows:write"
	ScopeTenantsRead             Scope = "tenants:read"
	ScopeTenantsWrite            Scope = "tenants:write"
	ScopeSettingsRead            Scope = "settings:read"
	ScopeSettingsWrite           Scope = "settings:write"
	ScopeSigningKeysRead         Scope = "signing-keys:read"
	ScopeSigningKeysWrite        Scope = "signing-keys:write"
	ScopeAuditRead               Scope = "audit:read"
	ScopeScimUsersRead           Scope = "scim:users:read"
	ScopeScimUsersWrite          Scope = "scim:users:write"
	ScopeScimGroupsRead          Scope = "scim:groups:read"
	ScopeScimGroupsWrite         Scope = "scim:groups:write"
)

var validScopes = map[Scope]struct{}{
	ScopeUsersRead: {}, ScopeUsersWrite: {}, ScopeGroupsRead: {}, ScopeGroupsWrite: {},
	ScopeAgentsRead: {}, ScopeAgentsWrite: {}, ScopeSessionsRead: {}, ScopeSessionsWrite: {},
	ScopeConsentsRead: {}, ScopeConsentsWrite: {}, ScopeLifecycleWorkflowsRead: {},
	ScopeLifecycleWorkflowsWrite: {}, ScopeTenantsRead: {}, ScopeTenantsWrite: {},
	ScopeSettingsRead: {}, ScopeSettingsWrite: {}, ScopeSigningKeysRead: {},
	ScopeSigningKeysWrite: {}, ScopeAuditRead: {}, ScopeScimUsersRead: {},
	ScopeScimUsersWrite: {}, ScopeScimGroupsRead: {}, ScopeScimGroupsWrite: {},
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

type TokenLiteral string

func ParseTokenLiteral(value string) (TokenLiteral, error) {
	if !strings.HasPrefix(value, TokenPrefix) {
		return "", ErrInvalidToken
	}
	raw := strings.TrimPrefix(value, TokenPrefix)
	if len(raw) != 64 {
		return "", ErrInvalidToken
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) != 32 {
		return "", ErrInvalidToken
	}
	return TokenLiteral(value), nil
}

func (t TokenLiteral) Hash() string {
	digest := sha256.Sum256([]byte(t))
	return hex.EncodeToString(digest[:])
}

type ApiToken struct {
	ID          string
	TenantID    string
	TokenHash   string
	Scopes      Scopes
	Description string
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

type Metadata struct {
	ID          string
	Description string
	Scopes      Scopes
	CreatedAt   time.Time
	ExpiresAt   *time.Time
}

func (t *ApiToken) Metadata() Metadata {
	return Metadata{
		ID: t.ID, Description: t.Description, Scopes: append(Scopes(nil), t.Scopes...),
		CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt,
	}
}

type Principal struct {
	TenantID string
	Scopes   Scopes
}
