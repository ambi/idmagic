package usecases

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
)

type oauthClientConfig = ports.OAuthClient

// コールバックの結果。戻り先の result に載せる。上流の値を含めない。
const (
	LinkResultLinked         = "linked"
	LinkResultInvalidState   = "invalid_state"
	LinkResultDenied         = "denied"
	LinkResultExchangeFailed = "exchange_failed"
)

const upstreamRevocationTimeout = 5 * time.Second

func randomToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashState(state string) []byte {
	sum := sha256.Sum256([]byte(state))
	return sum[:]
}

// StartLinkInput は連携の開始の入力である。UserID はブラウザーセッションの本人である。
type StartLinkInput struct {
	TenantID, ConnectorID, UserID, RedirectURI string
	Scopes                                     []string
}

// StartLink は連携を作り、上流の認可 URL を返す (REQ-TOKENVAULT-005)。
func StartLink(ctx context.Context, deps Deps, in StartLinkInput) (string, error) {
	connector, err := activeConnector(ctx, deps, in.TenantID, in.ConnectorID)
	if err != nil {
		return "", err
	}
	scopes := in.Scopes
	if len(scopes) == 0 {
		scopes = connector.AllowedScopes
	}
	for _, scope := range scopes {
		if !slices.Contains(connector.AllowedScopes, scope) {
			return "", ErrInvalidScope
		}
	}
	authorization, err := url.Parse(connector.AuthorizationEndpoint)
	if err != nil {
		return "", err
	}
	state, err := randomToken()
	if err != nil {
		return "", err
	}
	verifier, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := deps.Attempts.Save(ctx, &domain.ConnectionAttempt{
		ID: uuid.NewString(), TenantID: in.TenantID, ConnectorID: in.ConnectorID, UserID: in.UserID,
		StateHash: hashState(state), CodeVerifier: verifier, RequestedScopes: scopes,
		ExpiresAt: deps.now().Add(domain.AttemptTTL),
	}); err != nil {
		return "", err
	}
	challenge := sha256.Sum256([]byte(verifier))
	query := authorization.Query()
	query.Set("response_type", "code")
	query.Set("client_id", connector.ClientID)
	query.Set("redirect_uri", in.RedirectURI)
	query.Set("state", state)
	query.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	query.Set("code_challenge_method", "S256")
	if len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}
	authorization.RawQuery = query.Encode()
	return authorization.String(), nil
}

func activeConnector(ctx context.Context, deps Deps, tenantID, id string) (*domain.Connector, error) {
	connector, err := GetConnector(ctx, deps, tenantID, id)
	if err != nil {
		return nil, err
	}
	if connector.Status != domain.ConnectorActive {
		return nil, ErrConnectorNotFound
	}
	return connector, nil
}

// CompleteLinkInput はコールバックの入力である。UserID はセッションの本人で、なければ空である。
type CompleteLinkInput struct {
	TenantID, ConnectorID, UserID, State, Code, UpstreamError, RedirectURI string
}

// CompleteLink はコールバックを処理し、戻り先の結果を返す (REQ-TOKENVAULT-004)。error は
// 保存先の障害で、結果は exchange_failed である。
func CompleteLink(ctx context.Context, deps Deps, in CompleteLinkInput) (string, error) {
	if in.UserID == "" || in.State == "" {
		return LinkResultInvalidState, nil
	}
	now := deps.now()
	attempt, err := deps.Attempts.Consume(ctx, in.TenantID, in.ConnectorID, in.UserID, hashState(in.State), now)
	if err != nil {
		return LinkResultExchangeFailed, err
	}
	if attempt == nil {
		return LinkResultInvalidState, nil
	}
	if in.UpstreamError != "" {
		return LinkResultDenied, nil
	}
	connector, err := activeConnector(ctx, deps, in.TenantID, in.ConnectorID)
	if errors.Is(err, ErrConnectorNotFound) {
		return LinkResultInvalidState, nil
	}
	if err != nil {
		return LinkResultExchangeFailed, err
	}
	failed := &domain.TokenVaultConnectionLinkFailed{At: now, TenantID: in.TenantID, ConnectorID: in.ConnectorID, UserID: in.UserID}
	if in.Code == "" {
		deps.emit(failed)
		return LinkResultExchangeFailed, nil
	}
	response, err := deps.Upstream.ExchangeCode(ctx, oauthClient(connector), in.Code, attempt.CodeVerifier, in.RedirectURI)
	if err != nil {
		deps.emit(failed)
		return LinkResultExchangeFailed, nil //nolint:nilerr // 上流の拒否は利用者へ結果で知らせ、内容は載せない。
	}
	if err := deps.Connections.SaveLinked(ctx, linkedConnection(attempt, response, now)); err != nil {
		deps.emit(failed)
		return LinkResultExchangeFailed, err
	}
	deps.emit(&domain.TokenVaultConnectionLinked{At: now, TenantID: in.TenantID, ConnectorID: in.ConnectorID, UserID: in.UserID})
	return LinkResultLinked, nil
}

func linkedConnection(attempt *domain.ConnectionAttempt, response ports.TokenResponse, now time.Time) *domain.Connection {
	scopes := response.Scopes
	if scopes == nil {
		scopes = attempt.RequestedScopes
	}
	return &domain.Connection{
		ID: uuid.NewString(), TenantID: attempt.TenantID, ConnectorID: attempt.ConnectorID, UserID: attempt.UserID,
		Status: domain.ConnectionActive, GrantedScopes: scopes, AccessToken: response.AccessToken,
		RefreshToken: response.RefreshToken, ExpiresAt: expiry(response, now), CreatedAt: now, UpdatedAt: now,
	}
}

func expiry(response ports.TokenResponse, now time.Time) *time.Time {
	if response.ExpiresIn == nil {
		return nil
	}
	at := now.Add(*response.ExpiresIn)
	return &at
}

// MyConnection は本人から見たコネクタと接続である。Connection は接続がなければ nil である。
type MyConnection struct {
	Connector  *domain.Connector
	Connection *domain.Connection
}

// ListMyConnections は active のコネクタと、本人の接続を持つコネクタを返す (REQ-TOKENVAULT-006)。
func ListMyConnections(ctx context.Context, deps Deps, tenantID, userID string) ([]MyConnection, error) {
	connectors, err := deps.Connectors.List(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	connections, err := deps.Connections.ListByUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	result := make([]MyConnection, 0, len(connectors))
	for _, connector := range connectors {
		index := slices.IndexFunc(connections, func(c *domain.Connection) bool { return c.ConnectorID == connector.ID })
		var connection *domain.Connection
		if index >= 0 {
			connection = connections[index]
		}
		if connector.Status == domain.ConnectorActive || connection != nil {
			result = append(result, MyConnection{Connector: connector, Connection: connection})
		}
	}
	return result, nil
}

// Unlink は接続を消し、確定の後に上流の失効を一度だけ試す (REQ-TOKENVAULT-007)。
func Unlink(ctx context.Context, deps Deps, tenantID, connectorID, userID string) error {
	connection, err := deps.Connections.Delete(ctx, tenantID, connectorID, userID)
	if err != nil {
		return err
	}
	if connection == nil {
		return ErrConnectionNotFound
	}
	outcome := revokeUpstream(ctx, deps, connection)
	deps.emit(&domain.TokenVaultConnectionUnlinked{
		At: deps.now(), TenantID: tenantID, ConnectorID: connectorID, UserID: userID, UpstreamRevocation: outcome,
	})
	return nil
}

func revokeUpstream(ctx context.Context, deps Deps, connection *domain.Connection) string {
	token, hint := connection.RefreshToken, "refresh_token"
	if token == "" {
		token, hint = connection.AccessToken, "access_token"
	}
	connector, err := deps.Connectors.Find(ctx, connection.TenantID, connection.ConnectorID)
	if err != nil {
		return domain.UpstreamRevocationFailed
	}
	if connector == nil || connector.RevocationEndpoint == "" || token == "" {
		return domain.UpstreamRevocationNotConfigured
	}
	// 解除は確定しているので、呼び出し元の切断で失効の試行を止めない。
	revokeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), upstreamRevocationTimeout)
	defer cancel()
	if err := deps.Upstream.Revoke(revokeCtx, oauthClient(connector), token, hint); err != nil {
		return domain.UpstreamRevocationFailed
	}
	return domain.UpstreamRevocationSucceeded
}
