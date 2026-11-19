package handlers_http

import (
	"errors"
	"net/http"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type mcpToolDiscoveryMetadataResponse struct {
	Status               oauthdomain.MetadataStatus `json:"status"`
	URL                  string                     `json:"url,omitempty"`
	AuthorizationServers []string                   `json:"authorization_servers,omitempty"`
	ScopesSupported      []string                   `json:"scopes_supported,omitempty"`
	IssuerTrusted        *bool                      `json:"issuer_trusted,omitempty"`
}

type mcpToolDiscoveryToolsResponse struct {
	Status   oauthdomain.ToolsStatus `json:"status"`
	Count    *int                    `json:"count,omitempty"`
	Complete *bool                   `json:"complete,omitempty"`
}

type mcpToolDiscoveryResponse struct {
	ClientID string                           `json:"client_id"`
	Metadata mcpToolDiscoveryMetadataResponse `json:"metadata"`
	Tools    mcpToolDiscoveryToolsResponse    `json:"tools"`
}

// handleDiscoverAdminMcpTools は公開された MCP サーバーから PRM と一覧を取得する (REQ-OAUTH2-095)。
func (d Deps) handleDiscoverAdminMcpTools(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	principal, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	result, err := tokenusecases.DiscoverMcpServerTools(c.Request().Context(), tokenusecases.McpToolDiscoveryDeps{
		Servers: d.McpResourceServerRepo, Catalogs: d.McpToolCatalogRepo, Limiter: d.RateLimiter,
		Fetcher: d.McpToolDiscovery, Tokens: d.TokenIssuer, Emit: d.Emit,
	}, tokenusecases.McpToolDiscoveryRequest{
		TenantID: support.RequestTenantID(c), ServerID: c.Param("id"), ActorUserID: principal.ID,
		Issuer: support.RequestIssuer(c, d.Issuer),
	}, time.Now().UTC())
	var limited *tokenusecases.DiscoveryRateLimitedError
	switch {
	case errors.Is(err, tokenusecases.ErrMcpServerNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "resource_server_not_found", "The MCP resource server does not exist.")
	case errors.Is(err, tokenusecases.ErrMcpServerNotDiscoverable):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_resource_server", "Only an active self-hosted MCP resource server with a public HTTPS resource can be discovered.")
	case errors.As(err, &limited):
		return support.WriteRateLimited(c, limited.RetryAfterSeconds)
	case errors.Is(err, tokenusecases.ErrDiscoveryUnavailable):
		return support.WriteProblem(c, http.StatusServiceUnavailable, "temporarily_unavailable", "MCP tool discovery is temporarily unavailable.")
	case err != nil:
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toMcpToolDiscoveryResponse(result))
}

func toMcpToolDiscoveryResponse(result *tokenusecases.McpToolDiscoveryResult) mcpToolDiscoveryResponse {
	response := mcpToolDiscoveryResponse{
		ClientID: oauthdomain.McpToolDiscoveryClientID,
		Metadata: mcpToolDiscoveryMetadataResponse{Status: result.Metadata.Status, URL: result.Metadata.URL},
		Tools:    mcpToolDiscoveryToolsResponse{Status: result.Tools.Status},
	}
	if result.Metadata.Status == oauthdomain.MetadataFound {
		trusted := result.Metadata.IssuerTrusted
		response.Metadata.AuthorizationServers = result.Metadata.AuthorizationServers
		response.Metadata.ScopesSupported = result.Metadata.ScopesSupported
		response.Metadata.IssuerTrusted = &trusted
	}
	if result.Tools.Status == oauthdomain.ToolsFetched {
		count, complete := result.Tools.Count, result.Tools.Complete
		response.Tools.Count, response.Tools.Complete = &count, &complete
	}
	return response
}
