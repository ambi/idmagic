package domain

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/ambi/idmagic/backend/shared/spec"
)

// ResourceAccessTool は接続定義が承認したツールと業務操作の対応である。
// ResourceIDArgument があれば、既存の型と関係による対象別判定も必要になる。
type ResourceAccessTool struct {
	ID                   string `json:"id,omitempty"`
	AssignmentResourceID string `json:"assignment_resource_id,omitempty"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	ResourceIDArgument   string `json:"resource_id_argument,omitempty"`
}

func (r ResourceAccessRule) validateTool() error {
	tool := r.Tool
	if tool.ID == "" || tool.Name == "" || utf8.RuneCountInString(tool.Name) > 128 || strings.ContainsAny(tool.Name, ":#") || strings.TrimSpace(tool.DisplayName) == "" || utf8.RuneCountInString(tool.DisplayName) > 256 || utf8.RuneCountInString(tool.ResourceIDArgument) > 128 {
		return errors.New("invalid tool name, display name or target argument")
	}
	if r.ResourceType == spec.ResourceTypeMcpOperation {
		if r.Relation != spec.RelationMcpInvoke || tool.ResourceIDArgument != "" {
			return errors.New("built-in tool operations must use invoke without a target argument")
		}
	} else if tool.ResourceIDArgument == "" {
		return errors.New("resource-specific tools require a target argument")
	}
	return nil
}

func (m McpResourceServer) OperationResourceID(rule ResourceAccessRule) string {
	return m.ID + "/" + rule.Tool.ID
}

func (m McpResourceServer) ToolRuleFor(name string) (ResourceAccessRule, bool) {
	for _, r := range m.AccessRules {
		if r.Tool != nil && r.Tool.Name == name {
			return r, true
		}
	}
	return ResourceAccessRule{}, false
}

func (m McpResourceServer) HasOnlyToolRules() bool {
	if len(m.AccessRules) == 0 {
		return false
	}
	for _, r := range m.AccessRules {
		if r.Tool == nil {
			return false
		}
	}
	return true
}
