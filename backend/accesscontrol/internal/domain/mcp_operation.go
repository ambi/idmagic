package domain

import "github.com/ambi/idmagic/backend/shared/spec"

// McpOperationModel は通常の利用割当の固定スキーマである。
// 利用者のモデルの公開状態に依存させず、実際の付与は既存ストアから毎回読む。
func McpOperationModel(tenantID string) *AuthorizationModel {
	return &AuthorizationModel{ID: spec.ResourceTypeMcpOperation, TenantID: tenantID, Version: 1, ResourceTypes: []ResourceTypeDefinition{
		{Name: "user"},
		{Name: "agent"},
		// invoke は操作の割当と、対象を付けた ID への範囲を、invoke_any_target は全対象への範囲を表す。
		{Name: spec.ResourceTypeMcpOperation, Relations: []RelationDefinition{
			{Name: spec.RelationMcpInvoke, Rewrites: []RelationRewrite{{Kind: RewriteDirect, DirectSubjectTypes: []string{"user", "agent"}}}},
			{Name: spec.RelationMcpInvokeAnyTarget, Rewrites: []RelationRewrite{{Kind: RewriteDirect, DirectSubjectTypes: []string{"user", "agent"}}}},
		}},
	}}
}
