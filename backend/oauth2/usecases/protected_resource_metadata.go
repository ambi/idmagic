// Protected Resource Metadata (RFC 9728) のユースケース。配信内容は McpResourceServer
// 登録から導出し、手書きで独立に保守しない (ADR-011 と同方針、ADR-055)。
package usecases

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/ports"
)

type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

// BuildProtectedResourceMetadata は resource に対応する登録済み Active な
// McpResourceServer から Protected Resource Metadata を導出する。未登録・Disabled は
// fail-closed で invalid_target を返す。
func BuildProtectedResourceMetadata(
	ctx context.Context,
	repo ports.McpResourceServerRepository,
	tenantID, resource, issuer string,
) (*ProtectedResourceMetadata, error) {
	if repo == nil {
		return nil, NewOAuthError("invalid_target", "resource が登録されていません")
	}
	m, err := repo.FindByResource(ctx, tenantID, resource)
	if err != nil {
		return nil, err
	}
	if m == nil || !m.IsActive() {
		return nil, NewOAuthError("invalid_target", "resource が登録されていない、または無効です")
	}
	return &ProtectedResourceMetadata{
		Resource:               m.Resource,
		AuthorizationServers:   []string{issuer},
		ScopesSupported:        m.Scopes,
		BearerMethodsSupported: []string{"header"},
	}, nil
}
