// Resource Indicators (RFC 8707) の共通検証。Authorize / PushAuthorizationRequest /
// Token(authorization_code redemption) / Token(token-exchange) から呼ばれる (ADR-055)。
package usecases

import (
	"context"
	"slices"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

// ResolveResourceIndicator は resource パラメータを検証し、対応する Active な
// McpResourceServer を返す。
//
//   - resources が空 (resource 未指定) の場合は (nil, nil) を返し、呼び出し側は
//     従来どおり client_id を audience とする発行を続ける (後方互換)。
//   - resources が複数、または tenant 内に登録された Active な McpResourceServer が
//     見つからない場合は fail-closed で invalid_target を返す (ADR-055 決定3)。
//   - requestedScopes が resource の許可 scope (allowlist) の部分集合でない場合は
//     invalid_scope を返す。
func ResolveResourceIndicator(
	ctx context.Context,
	repo ports.McpResourceServerRepository,
	tenantID string,
	resources []string,
	requestedScopes []string,
) (*domain.McpResourceServer, error) {
	values := nonEmpty(resources)
	if len(values) == 0 {
		return nil, nil //nolint:nilnil // resource 未指定は「audience 限定なし」を表す正常系
	}
	if len(values) > 1 {
		return nil, NewOAuthError("invalid_target", "resource は 1 個のみ指定できます (1 token = 1 resource)")
	}
	resource := values[0]
	if repo == nil {
		return nil, NewOAuthError("invalid_target", "resource が登録されていません")
	}
	mcp, err := repo.FindByResource(ctx, tenantID, resource)
	if err != nil {
		return nil, err
	}
	if mcp == nil || !mcp.IsActive() {
		return nil, NewOAuthError("invalid_target", "resource が登録されていない、または無効です")
	}
	for _, scope := range requestedScopes {
		if !slices.Contains(mcp.Scopes, scope) {
			return nil, NewOAuthError("invalid_scope", "resource の許可 scope を超える要求です")
		}
	}
	return mcp, nil
}
