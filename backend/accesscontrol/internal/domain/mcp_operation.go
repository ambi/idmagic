package domain

import "github.com/ambi/idmagic/backend/shared/spec"

// McpOperationModel は通常の利用割当の固定スキーマである。
// 利用者のモデルの公開状態に依存させず、実際の付与は既存ストアから毎回読む。
func McpOperationModel(tenantID string) *AuthorizationModel {
	return &AuthorizationModel{ID: spec.ResourceTypeMcpOperation, TenantID: tenantID, Version: 1, ResourceTypes: []ResourceTypeDefinition{
		{Name: "user"},
		{Name: "agent"},
		{Name: spec.ResourceTypeMcpOperation, Relations: []RelationDefinition{{Name: spec.RelationMcpInvoke, Rewrites: []RelationRewrite{{Kind: RewriteDirect, DirectSubjectTypes: []string{"user", "agent"}}}}}},
	}}
}
