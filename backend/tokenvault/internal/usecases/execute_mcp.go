package usecases

import (
	"context"
	"net/http"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// ExecuteMCP は検証済みの呼び出し元にも Vault の認可を適用する。
// DPoP の再検証と、上流の 401 を理由にした業務要求の再送はしない。
func ExecuteMCP(ctx context.Context, deps Deps, upstream ports.McpUpstreamClient, in ExecuteInput, token domain.CallerToken) (*http.Response, error) {
	target, err := domain.ParseExecutionTarget(in.URL)
	if err != nil {
		return nil, err
	}
	if len(in.Body) > domain.MaxRequestBodyBytes {
		return nil, ErrRequestTooLarge
	}
	connector, rule, err := authorize(ctx, deps, in, token, target)
	if err != nil {
		return nil, err
	}
	connection, err := usableConnection(ctx, deps, connector, token.Subject, 0)
	if err != nil {
		return nil, translateRefreshError(err)
	}
	response, err := upstream.OpenMCP(ctx, ports.OutboundRequest{Method: http.MethodPost, URL: in.URL, Headers: in.Headers, Body: in.Body, AccessToken: connection.AccessToken})
	status := 0
	if response != nil {
		status = response.StatusCode
	}
	now := deps.now()
	deps.emit(&domain.TokenVaultExecutionPerformed{
		At: now, TenantID: in.TenantID, ConnectorID: in.ConnectorID, UserID: token.Subject, AgentID: token.ActingAgent(),
		RuleID: rule.ID, Method: in.Method, Origin: domain.Origin(target), UpstreamStatus: status,
	})
	if err != nil {
		return nil, err
	}
	// 表示用の最終利用時刻の失敗を、完了した上流の作用の再試行へ変えない。
	_ = deps.Connections.TouchLastUsed(ctx, in.TenantID, connection.ID, now)
	return response, nil
}
