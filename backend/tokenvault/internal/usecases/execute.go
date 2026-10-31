package usecases

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// ExecuteInput は代理実行の入力である。
type ExecuteInput struct {
	TenantID, ConnectorID, Method, URL string
	Headers                            map[string]string
	Body                               []byte
	Credential                         ports.CallerCredential
}

// Execute はモデルの論理積を評価し、満たせば保管したトークンで上流へ送る (REQ-TOKENVAULT-008)。
// 上流のトークンは戻り値に現れない。
func Execute(ctx context.Context, deps Deps, in ExecuteInput) (ports.OutboundResponse, error) {
	token, err := deps.Verifier.Verify(ctx, in.Credential)
	if err != nil {
		return ports.OutboundResponse{}, ErrTemporarilyUnavailable
	}
	if token == nil {
		return ports.OutboundResponse{}, ErrInvalidToken
	}
	target, err := domain.ParseExecutionTarget(in.URL)
	if err != nil {
		return ports.OutboundResponse{}, err
	}
	if err := domain.CheckForwardedHeaders(in.Headers); err != nil {
		return ports.OutboundResponse{}, err
	}
	if len(in.Body) > domain.MaxRequestBodyBytes {
		return ports.OutboundResponse{}, ErrRequestTooLarge
	}
	connector, rule, err := authorize(ctx, deps, in, *token, target)
	if err != nil {
		return ports.OutboundResponse{}, err
	}
	return forward(ctx, deps, in, *token, connector, rule, target)
}

func authorize(
	ctx context.Context, deps Deps, in ExecuteInput, token domain.CallerToken, target *url.URL,
) (*domain.Connector, domain.ExecutionRule, error) {
	deny := func(reason string, err error) (*domain.Connector, domain.ExecutionRule, error) {
		deps.emit(&domain.TokenVaultExecutionDenied{
			At: deps.now(), TenantID: in.TenantID, ConnectorID: in.ConnectorID,
			UserID: token.Subject, AgentID: token.ActingAgent(), Reason: reason,
		})
		return nil, domain.ExecutionRule{}, err
	}
	if reason := domain.AuthorizeCaller(token, in.ConnectorID, in.Method, target); reason != "" {
		return deny(reason, ErrAccessDenied)
	}
	connector, err := activeConnector(ctx, deps, in.TenantID, in.ConnectorID)
	if errors.Is(err, ErrConnectorNotFound) {
		return deny("connector_unavailable", err)
	}
	if err != nil {
		return nil, domain.ExecutionRule{}, ErrTemporarilyUnavailable
	}
	rule, matched := connector.MatchRule(in.Method, target)
	if !matched {
		return deny(domain.ReasonRuleMismatch, ErrAccessDenied)
	}
	connection, err := deps.Connections.Find(ctx, in.TenantID, in.ConnectorID, token.Subject)
	if err != nil {
		return nil, domain.ExecutionRule{}, ErrTemporarilyUnavailable
	}
	switch {
	case connection == nil:
		return deny("connection_not_found", ErrConnectionNotFound)
	case connection.Status != domain.ConnectionActive:
		return deny("connection_reauthorization_required", ErrReauthorizationRequired)
	case !connection.HasScopes(rule.RequiredScopes):
		return deny(domain.ReasonScopeInsufficient, ErrAccessDenied)
	}
	return connector, rule, nil
}

func forward(
	ctx context.Context, deps Deps, in ExecuteInput, token domain.CallerToken,
	connector *domain.Connector, rule domain.ExecutionRule, target *url.URL,
) (ports.OutboundResponse, error) {
	connection, err := usableConnection(ctx, deps, connector, token.Subject, 0)
	if err != nil {
		return ports.OutboundResponse{}, translateRefreshError(err)
	}
	request := ports.OutboundRequest{Method: in.Method, URL: in.URL, Headers: in.Headers, Body: in.Body, AccessToken: connection.AccessToken}
	response, sendErr := deps.Upstream.Send(ctx, request)
	if sendErr == nil && response.Status == http.StatusUnauthorized && connection.RefreshToken != "" {
		// 上流が拒否した要求は処理されていないので、メソッドを問わず一度だけ再送できる (REQ-TOKENVAULT-014)。
		connection, err = usableConnection(ctx, deps, connector, token.Subject, connection.TokenVersion)
		if err != nil {
			return ports.OutboundResponse{}, translateRefreshError(err)
		}
		request.AccessToken = connection.AccessToken
		response, sendErr = deps.Upstream.Send(ctx, request)
	}
	now := deps.now()
	status := response.Status
	if sendErr != nil {
		status = 0
	}
	deps.emit(&domain.TokenVaultExecutionPerformed{
		At: now, TenantID: in.TenantID, ConnectorID: in.ConnectorID, UserID: token.Subject, AgentID: token.ActingAgent(),
		RuleID: rule.ID, Method: in.Method, Origin: domain.Origin(target), UpstreamStatus: status,
	})
	if sendErr != nil {
		return ports.OutboundResponse{}, sendErr
	}
	if err := deps.Connections.TouchLastUsed(ctx, in.TenantID, connection.ID, now); err != nil {
		// 上流の作用は確定している。表示用の値のために結果を失わせない。
		slog.WarnContext(ctx, "token vault could not record last use", "connector_id", in.ConnectorID, "error", err)
	}
	return response, nil
}

func translateRefreshError(err error) error {
	switch {
	case errors.Is(err, ErrReauthorizationRequired), errors.Is(err, ErrConnectionNotFound), errors.Is(err, ErrTemporarilyUnavailable):
		return err
	default:
		return ErrTemporarilyUnavailable
	}
}
