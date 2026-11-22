package usecases

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	tokenvaultdomain "github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	tokenvaultports "github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

// ExecuteInput は代理実行の入力である。
type ExecuteInput struct {
	TenantID, ConnectorID, Method, URL string
	Headers                            map[string]string
	Body                               []byte
	Credential                         tokenvaultports.CallerCredential
}

// Execute はモデルの論理積を評価し、満たせば保管したトークンで上流へ送る (REQ-TOKENVAULT-008)。
// 上流のトークンは戻り値に現れない。
func Execute(ctx context.Context, deps Deps, in ExecuteInput) (tokenvaultports.OutboundResponse, error) {
	token, err := deps.Verifier.Verify(ctx, in.Credential)
	if err != nil {
		return tokenvaultports.OutboundResponse{}, ErrTemporarilyUnavailable
	}
	if token == nil {
		return tokenvaultports.OutboundResponse{}, ErrInvalidToken
	}
	target, err := tokenvaultdomain.ParseExecutionTarget(in.URL)
	if err != nil {
		return tokenvaultports.OutboundResponse{}, err
	}
	if err := tokenvaultdomain.CheckForwardedHeaders(in.Headers); err != nil {
		return tokenvaultports.OutboundResponse{}, err
	}
	if len(in.Body) > tokenvaultdomain.MaxRequestBodyBytes {
		return tokenvaultports.OutboundResponse{}, ErrRequestTooLarge
	}
	connector, rule, err := authorize(ctx, deps, in, *token, target)
	if err != nil {
		return tokenvaultports.OutboundResponse{}, err
	}
	return forward(ctx, deps, in, *token, connector, rule, target)
}

func authorize(
	ctx context.Context, deps Deps, in ExecuteInput, token tokenvaultdomain.CallerToken, target *url.URL,
) (*tokenvaultdomain.Connector, tokenvaultdomain.ExecutionRule, error) {
	deny := func(reason string, err error) (*tokenvaultdomain.Connector, tokenvaultdomain.ExecutionRule, error) {
		deps.emit(&tokenvaultdomain.TokenVaultExecutionDenied{
			At: deps.now(), TenantID: in.TenantID, ConnectorID: in.ConnectorID,
			UserID: token.Subject, AgentID: token.ActingAgent(), Reason: reason,
		})
		return nil, tokenvaultdomain.ExecutionRule{}, err
	}
	if reason := tokenvaultdomain.AuthorizeCaller(token, in.ConnectorID, in.Method, target); reason != "" {
		return deny(reason, ErrAccessDenied)
	}
	connector, err := activeConnector(ctx, deps, in.TenantID, in.ConnectorID)
	if errors.Is(err, ErrConnectorNotFound) {
		return deny("connector_unavailable", err)
	}
	if err != nil {
		return nil, tokenvaultdomain.ExecutionRule{}, ErrTemporarilyUnavailable
	}
	rule, matched := connector.MatchRule(in.Method, target)
	if !matched {
		return deny(tokenvaultdomain.ReasonRuleMismatch, ErrAccessDenied)
	}
	connection, err := deps.Connections.Find(ctx, in.TenantID, in.ConnectorID, token.Subject)
	if err != nil {
		return nil, tokenvaultdomain.ExecutionRule{}, ErrTemporarilyUnavailable
	}
	switch {
	case connection == nil:
		return deny("connection_not_found", ErrConnectionNotFound)
	case connection.Status != tokenvaultdomain.ConnectionActive:
		return deny("connection_reauthorization_required", ErrReauthorizationRequired)
	case !connection.HasScopes(rule.RequiredScopes):
		return deny(tokenvaultdomain.ReasonScopeInsufficient, ErrAccessDenied)
	}
	return connector, rule, nil
}

func forward(
	ctx context.Context, deps Deps, in ExecuteInput, token tokenvaultdomain.CallerToken,
	connector *tokenvaultdomain.Connector, rule tokenvaultdomain.ExecutionRule, target *url.URL,
) (tokenvaultports.OutboundResponse, error) {
	connection, err := usableConnection(ctx, deps, connector, token.Subject, 0)
	if err != nil {
		return tokenvaultports.OutboundResponse{}, translateRefreshError(err)
	}
	request := tokenvaultports.OutboundRequest{Method: in.Method, URL: in.URL, Headers: in.Headers, Body: in.Body, AccessToken: connection.AccessToken}
	response, sendErr := deps.Upstream.Send(ctx, request)
	if sendErr == nil && response.Status == http.StatusUnauthorized && connection.RefreshToken != "" {
		// 上流が拒否した要求は処理されていないので、メソッドを問わず一度だけ再送できる (REQ-TOKENVAULT-014)。
		connection, err = usableConnection(ctx, deps, connector, token.Subject, connection.TokenVersion)
		if err != nil {
			return tokenvaultports.OutboundResponse{}, translateRefreshError(err)
		}
		request.AccessToken = connection.AccessToken
		response, sendErr = deps.Upstream.Send(ctx, request)
	}
	now := deps.now()
	status := response.Status
	if sendErr != nil {
		status = 0
	}
	deps.emit(&tokenvaultdomain.TokenVaultExecutionPerformed{
		At: now, TenantID: in.TenantID, ConnectorID: in.ConnectorID, UserID: token.Subject, AgentID: token.ActingAgent(),
		RuleID: rule.ID, Method: in.Method, Origin: tokenvaultdomain.Origin(target), UpstreamStatus: status,
	})
	if sendErr != nil {
		return tokenvaultports.OutboundResponse{}, sendErr
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
