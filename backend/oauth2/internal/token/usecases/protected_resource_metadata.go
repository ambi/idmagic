// Protected Resource Metadata (RFC 9728) のユースケース。配信内容は McpResourceServer
// 登録から導出し、手書きで独立に保守しない。
package usecases

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/ports"
)

type ProtectedResourceMetadata struct {
	Resource                      string   `json:"resource"`
	AuthorizationServers          []string `json:"authorization_servers"`
	ScopesSupported               []string `json:"scopes_supported"`
	BearerMethodsSupported        []string `json:"bearer_methods_supported"`
	DPoPSigningAlgValuesSupported []string `json:"dpop_signing_alg_values_supported,omitempty"`
}

// BuiltinResource は、認可サーバー自身が保護する realm の API について、管理発行
// トークンの所有者が組み立て地点から渡す情報である。
type BuiltinResource struct {
	// Scopes は resource を省いた Protected Resource Metadata の scopes_supported に載せる。
	Scopes []string
	// RevocationClientID は、クライアント認証なしで /revoke を呼べる組み込みの公開
	// クライアントの ID である。空なら例外を設けない。
	RevocationClientID string
}

// BuiltinProtectedResourceMetadata は、resource を省いた要求に返す realm の API の
// Protected Resource Metadata を作る。
func BuiltinProtectedResourceMetadata(builtin BuiltinResource, issuer string) *ProtectedResourceMetadata {
	return &ProtectedResourceMetadata{
		Resource: issuer, AuthorizationServers: []string{issuer},
		ScopesSupported: builtin.Scopes, BearerMethodsSupported: []string{"header"},
		DPoPSigningAlgValuesSupported: []string{"ES256", "RS256", "PS256"},
	}
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
		return nil, NewOAuthError("invalid_target", "The resource is not registered.")
	}
	m, err := repo.FindByResource(ctx, tenantID, resource)
	if err != nil {
		return nil, err
	}
	if m == nil || !m.IsActive() {
		return nil, NewOAuthError("invalid_target", "The resource is not registered or is disabled.")
	}
	return &ProtectedResourceMetadata{
		Resource:               m.Resource,
		AuthorizationServers:   []string{issuer},
		ScopesSupported:        m.Scopes,
		BearerMethodsSupported: []string{"header"},
	}, nil
}
