// Package handlers_http は TokenVault の HTTP の入口である。
package handlers_http

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tokenvault/internal/domain"
	"github.com/ambi/idmagic/backend/tokenvault/internal/ports"
	"github.com/ambi/idmagic/backend/tokenvault/internal/usecases"

	"github.com/labstack/echo/v5"
)

// base64 の膨張と JSON の項目を見込んだ、復号の前の要求本文の上限である。
const maxExecuteEnvelopeBytes = domain.MaxRequestBodyBytes/3*4 + 64<<10

// Deps は TokenVault の HTTP の依存である。
type Deps struct {
	support.Deps
	*support.Authenticator
	UseCases usecases.Deps
}

// RegisterRoutes は TokenVault の経路を登録する。
func RegisterRoutes(g *echo.Group, d Deps) {
	g.POST("/api/admin/v1/token-vault/connectors", d.handleCreateConnector)
	g.GET("/api/admin/v1/token-vault/connectors", d.handleListConnectors)
	g.GET("/api/admin/v1/token-vault/connectors/:id", d.handleGetConnector)
	g.PATCH("/api/admin/v1/token-vault/connectors/:id", d.handleUpdateConnector)
	g.DELETE("/api/admin/v1/token-vault/connectors/:id", d.handleDeleteConnector)
	g.GET("/api/account/v1/token-vault/connections", d.handleListMyConnections)
	g.POST("/api/account/v1/token-vault/connections/:connector_id/authorize", d.handleStartLink)
	g.DELETE("/api/account/v1/token-vault/connections/:connector_id", d.handleUnlink)
	g.GET("/api/token-vault/v1/callback/:connector_id", d.handleCallback)
	g.POST("/api/token-vault/v1/connectors/:connector_id/execute", d.handleExecute)
}

type connectorResponse struct {
	ID                      string                         `json:"id"`
	TenantID                string                         `json:"tenant_id"`
	Name                    string                         `json:"name"`
	Status                  domain.ConnectorStatus         `json:"status"`
	AuthorizationEndpoint   string                         `json:"authorization_endpoint"`
	TokenEndpoint           string                         `json:"token_endpoint"`
	RevocationEndpoint      string                         `json:"revocation_endpoint,omitempty"`
	ClientID                string                         `json:"client_id"`
	TokenEndpointAuthMethod domain.TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
	ClientSecretConfigured  bool                           `json:"client_secret_configured"`
	AllowedScopes           []string                       `json:"allowed_scopes"`
	Rules                   []domain.ExecutionRule         `json:"rules"`
	CreatedAt               time.Time                      `json:"created_at"`
	UpdatedAt               time.Time                      `json:"updated_at"`
}

func toConnectorResponse(c *domain.Connector) connectorResponse {
	return connectorResponse{
		ID: c.ID, TenantID: c.TenantID, Name: c.Name, Status: c.Status, AuthorizationEndpoint: c.AuthorizationEndpoint,
		TokenEndpoint: c.TokenEndpoint, RevocationEndpoint: c.RevocationEndpoint, ClientID: c.ClientID,
		TokenEndpointAuthMethod: c.TokenEndpointAuthMethod, ClientSecretConfigured: c.ClientSecret != "",
		AllowedScopes: c.AllowedScopes, Rules: c.Rules, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

type connectorCreateRequest struct {
	Name                    string                         `json:"name"`
	AuthorizationEndpoint   string                         `json:"authorization_endpoint"`
	TokenEndpoint           string                         `json:"token_endpoint"`
	RevocationEndpoint      string                         `json:"revocation_endpoint"`
	ClientID                string                         `json:"client_id"`
	TokenEndpointAuthMethod domain.TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
	ClientSecret            string                         `json:"client_secret"`
	AllowedScopes           []string                       `json:"allowed_scopes"`
	Rules                   []domain.ExecutionRule         `json:"rules"`
}

func (d Deps) writeConnectorError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidConnector):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The connector definition is invalid.")
	case errors.Is(err, ports.ErrConnectorConflict):
		return support.WriteProblem(c, http.StatusConflict, "connector_conflict", "The connector conflicts with an existing one.")
	case errors.Is(err, usecases.ErrConnectorNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "connector_not_found", "The connector does not exist.")
	default:
		return err
	}
}

func (d Deps) handleCreateConnector(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req connectorCreateRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	connector, err := usecases.CreateConnector(c.Request().Context(), d.UseCases, domain.Connector{
		TenantID: support.RequestTenantID(c), Name: req.Name, AuthorizationEndpoint: req.AuthorizationEndpoint,
		TokenEndpoint: req.TokenEndpoint, RevocationEndpoint: req.RevocationEndpoint, ClientID: req.ClientID,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod, ClientSecret: req.ClientSecret,
		AllowedScopes: req.AllowedScopes, Rules: req.Rules,
	})
	if err != nil {
		return d.writeConnectorError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toConnectorResponse(connector))
}

func (d Deps) handleListConnectors(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	connectors, err := d.UseCases.Connectors.List(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	items := make([]connectorResponse, len(connectors))
	for i, connector := range connectors {
		items[i] = toConnectorResponse(connector)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"items": items})
}

func (d Deps) handleGetConnector(c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	connector, err := usecases.GetConnector(c.Request().Context(), d.UseCases, support.RequestTenantID(c), c.Param("id"))
	if err != nil {
		return d.writeConnectorError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toConnectorResponse(connector))
}

type connectorUpdateRequest struct {
	Name                    *string                         `json:"name"`
	Status                  *domain.ConnectorStatus         `json:"status"`
	AuthorizationEndpoint   *string                         `json:"authorization_endpoint"`
	TokenEndpoint           *string                         `json:"token_endpoint"`
	RevocationEndpoint      *string                         `json:"revocation_endpoint"`
	ClientID                *string                         `json:"client_id"`
	TokenEndpointAuthMethod *domain.TokenEndpointAuthMethod `json:"token_endpoint_auth_method"`
	ClientSecret            *string                         `json:"client_secret"`
	AllowedScopes           *[]string                       `json:"allowed_scopes"`
	Rules                   *[]domain.ExecutionRule         `json:"rules"`
}

func (d Deps) handleUpdateConnector(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var req connectorUpdateRequest
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	connector, err := usecases.UpdateConnector(c.Request().Context(), d.UseCases, support.RequestTenantID(c), c.Param("id"), usecases.ConnectorChanges{
		Name: req.Name, Status: req.Status, AuthorizationEndpoint: req.AuthorizationEndpoint, TokenEndpoint: req.TokenEndpoint,
		RevocationEndpoint: req.RevocationEndpoint, ClientID: req.ClientID, TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		ClientSecret: req.ClientSecret, AllowedScopes: req.AllowedScopes, Rules: req.Rules,
	})
	if err != nil {
		return d.writeConnectorError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toConnectorResponse(connector))
}

func (d Deps) handleDeleteConnector(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := usecases.DeleteConnector(c.Request().Context(), d.UseCases, support.RequestTenantID(c), c.Param("id")); err != nil {
		return d.writeConnectorError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) requireAccountSub(c *echo.Context) (string, error) {
	authn, err := d.Authenticate(c)
	if err != nil {
		return "", err
	}
	if authn == nil || authn.Pending() {
		return "", support.ErrAdminAuthenticationRequired
	}
	return authn.Subject(), nil
}

func writeAccountSubError(c *echo.Context, err error) error {
	if handled, result := support.WriteAccessTokenError(c, err); handled {
		return result
	}
	if errors.Is(err, support.ErrAdminAuthenticationRequired) {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "Authentication is required.")
	}
	return err
}

type myConnectionResponse struct {
	ConnectorID     string                  `json:"connector_id"`
	ConnectorName   string                  `json:"connector_name"`
	ConnectorStatus domain.ConnectorStatus  `json:"connector_status"`
	Status          domain.ConnectionStatus `json:"status,omitempty"`
	GrantedScopes   []string                `json:"granted_scopes"`
	LastUsedAt      *time.Time              `json:"last_used_at,omitempty"`
}

func (d Deps) handleListMyConnections(c *echo.Context) error {
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	mine, err := usecases.ListMyConnections(c.Request().Context(), d.UseCases, support.RequestTenantID(c), sub)
	if err != nil {
		return err
	}
	items := make([]myConnectionResponse, len(mine))
	for i, entry := range mine {
		items[i] = myConnectionResponse{
			ConnectorID: entry.Connector.ID, ConnectorName: entry.Connector.Name,
			ConnectorStatus: entry.Connector.Status, GrantedScopes: []string{},
		}
		if connection := entry.Connection; connection != nil {
			items[i].Status, items[i].GrantedScopes, items[i].LastUsedAt = connection.Status, connection.GrantedScopes, connection.LastUsedAt
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"items": items})
}

// トークンで連携、解除できると、account:write を持つ Agent が自分で接続を作れる (REQ-TOKENVAULT-005)。
// ブラウザーの検証はトークンを素通しするので、先に拒否する。
func refuseTokenAuthorization(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusForbidden, "access_denied", "This operation requires a browser session.")
}

func hasTokenAuthorization(c *echo.Context) bool {
	return c.Request().Header.Get("Authorization") != ""
}

func (d Deps) callbackURI(c *echo.Context, connectorID string) string {
	return support.TenantURL(c, "/api/token-vault/v1/callback/"+url.PathEscape(connectorID), d.Issuer)
}

func (d Deps) handleStartLink(c *echo.Context) error {
	if hasTokenAuthorization(c) {
		return refuseTokenAuthorization(c)
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	var req struct {
		Scopes []string `json:"scopes"`
	}
	if err := support.DecodeJSON(c.Request(), &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	connectorID := c.Param("connector_id")
	authorizationURL, err := usecases.StartLink(c.Request().Context(), d.UseCases, usecases.StartLinkInput{
		TenantID: support.RequestTenantID(c), ConnectorID: connectorID, UserID: sub,
		RedirectURI: d.callbackURI(c, connectorID), Scopes: req.Scopes,
	})
	switch {
	case errors.Is(err, usecases.ErrInvalidScope):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_scope", "The connector does not allow the requested scope.")
	case errors.Is(err, usecases.ErrConnectorNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "connector_not_found", "The connector does not exist.")
	case err != nil:
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"authorization_url": authorizationURL})
}

func (d Deps) handleUnlink(c *echo.Context) error {
	if hasTokenAuthorization(c) {
		return refuseTokenAuthorization(c)
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAccountSub(c)
	if err != nil {
		return writeAccountSubError(c, err)
	}
	err = usecases.Unlink(c.Request().Context(), d.UseCases, support.RequestTenantID(c), c.Param("connector_id"), sub)
	if errors.Is(err, usecases.ErrConnectionNotFound) {
		return support.WriteProblem(c, http.StatusNotFound, "connection_not_found", "The connection does not exist.")
	}
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// handleCallback は上流からの戻りを処理する。セッションの本人を連携の本人と照合するので、
// Authorization は読まない。
func (d Deps) handleCallback(c *echo.Context) error {
	connectorID := c.Param("connector_id")
	userID := ""
	if !hasTokenAuthorization(c) {
		if sub, err := d.requireAccountSub(c); err == nil {
			userID = sub
		}
	}
	result, err := usecases.CompleteLink(c.Request().Context(), d.UseCases, usecases.CompleteLinkInput{
		TenantID: support.RequestTenantID(c), ConnectorID: connectorID, UserID: userID,
		State: c.QueryParam("state"), Code: c.QueryParam("code"), UpstreamError: c.QueryParam("error"),
		RedirectURI: d.callbackURI(c, connectorID),
	})
	if err != nil {
		slog.ErrorContext(c.Request().Context(), "token vault callback failed", "connector_id", connectorID, "error", err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Redirect(http.StatusFound, support.TenantURL(c, "/account/connections?result="+result, d.Issuer))
}

type executeRequest struct {
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Headers    map[string]string `json:"headers"`
	BodyBase64 string            `json:"body_base64"`
}

func (d Deps) handleExecute(c *echo.Context) error {
	raw, scheme := bearerOrDPoP(c)
	if raw == "" {
		return writeInvalidToken(c)
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Response(), c.Request().Body, maxExecuteEnvelopeBytes))
	if err != nil {
		return support.WriteProblem(c, http.StatusRequestEntityTooLarge, "request_too_large", "The request body is too large.")
	}
	var req executeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	decoded, err := base64.StdEncoding.DecodeString(req.BodyBase64)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "body_base64 is invalid.")
	}
	response, err := usecases.Execute(c.Request().Context(), d.UseCases, usecases.ExecuteInput{
		TenantID: support.RequestTenantID(c), ConnectorID: c.Param("connector_id"),
		Method: req.Method, URL: req.URL, Headers: req.Headers, Body: decoded,
		Credential: ports.CallerCredential{
			Token: raw, Scheme: scheme, DPoPProof: c.Request().Header.Get("DPoP"),
			Method: c.Request().Method, HTU: support.RequestHTU(c, d.Issuer),
			Resource: support.TenantURL(c, "/api/token-vault/v1", d.Issuer),
		},
	})
	if err != nil {
		return writeExecuteError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"status": response.Status, "content_type": response.ContentType,
		"body_base64": base64.StdEncoding.EncodeToString(response.Body),
	})
}

func bearerOrDPoP(c *echo.Context) (token, scheme string) {
	header := c.Request().Header.Get("Authorization")
	for _, candidate := range []string{"bearer", "dpop"} {
		prefix := candidate + " "
		if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
			return strings.TrimSpace(header[len(prefix):]), candidate
		}
	}
	return "", ""
}

func writeInvalidToken(c *echo.Context) error {
	support.SetBearerChallenge(c, "invalid_token", "")
	return support.WriteProblem(c, http.StatusUnauthorized, "invalid_token", "The access token is invalid.")
}

func writeExecuteError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, usecases.ErrInvalidToken):
		return writeInvalidToken(c)
	case errors.Is(err, domain.ErrInvalidExecutionRequest):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The request is invalid.")
	case errors.Is(err, usecases.ErrRequestTooLarge):
		return support.WriteProblem(c, http.StatusRequestEntityTooLarge, "request_too_large", "The request body is too large.")
	case errors.Is(err, usecases.ErrAccessDenied):
		return support.WriteProblem(c, http.StatusForbidden, "connector_access_denied", "The request is not permitted.")
	case errors.Is(err, usecases.ErrConnectorNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "connector_not_found", "The connector does not exist.")
	case errors.Is(err, usecases.ErrConnectionNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "connection_not_found", "The connection does not exist.")
	case errors.Is(err, usecases.ErrReauthorizationRequired):
		return support.WriteProblem(c, http.StatusConflict, "connection_reauthorization_required", "The connection must be linked again.")
	case errors.Is(err, ports.ErrUpstreamUnreachable):
		return support.WriteProblem(c, http.StatusBadGateway, "upstream_unreachable", "The upstream could not be reached.")
	case errors.Is(err, ports.ErrUpstreamResponseTooLarge):
		return support.WriteProblem(c, http.StatusBadGateway, "upstream_response_too_large", "The upstream response is too large.")
	default:
		return support.WriteProblem(c, http.StatusServiceUnavailable, "temporarily_unavailable", "The request cannot be processed now.")
	}
}
