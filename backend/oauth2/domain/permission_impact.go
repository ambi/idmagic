package domain

import (
	"errors"
	"slices"
)

func (m McpResourceServer) validateScopeHints() error {
	for scope, hint := range m.ScopeHints {
		if !slices.Contains(m.Scopes, scope) {
			return errors.New("scope_hints must name registered scopes")
		}
		if hint == nil {
			return errors.New("scope_hints must contain complete impacts")
		}
		if err := hint.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func CloneScopeHints(hints map[string]*PermissionImpact) map[string]*PermissionImpact {
	if hints == nil {
		return nil
	}
	out := make(map[string]*PermissionImpact, len(hints))
	for scope, hint := range hints {
		out[scope] = hint.Clone()
	}
	return out
}

// ScopeHints はリソースを特定しない権限では同名の全登録を保守的に集約する。
// 未登録を一つでも含むスコープにはヒントを返さない。
func ScopeHints(servers []*McpResourceServer, resource string) map[string]*PermissionImpact {
	out := map[string]*PermissionImpact{}
	unknown := map[string]bool{}
	for _, server := range servers {
		if !server.IsActive() || (resource != "" && server.Resource != resource) {
			continue
		}
		for _, scope := range server.Scopes {
			hint := server.ScopeHints[scope]
			if hint == nil || hint.Validate() != nil {
				unknown[scope] = true
				continue
			}
			current := out[scope]
			if current == nil {
				out[scope] = hint.Clone()
				continue
			}
			if impactRank(hint.Changes) > impactRank(current.Changes) {
				current.Changes = hint.Changes
			}
			*current.PersonalData = *current.PersonalData || *hint.PersonalData
		}
	}
	for scope := range unknown {
		delete(out, scope)
	}
	return out
}

func impactRank(changes string) int {
	switch changes {
	case "none":
		return 0
	case "reversible":
		return 1
	default:
		return 2
	}
}
