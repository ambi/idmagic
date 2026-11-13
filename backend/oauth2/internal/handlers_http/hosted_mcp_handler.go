package handlers_http

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/internal/token/usecases"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/protocol/mcp"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/safehttp"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"

	"github.com/labstack/echo/v5"
)

const (
	hostedMcpRequestLimit  = 1 << 20
	hostedMcpResponseLimit = 16 << 20
	hostedMcpTimeout       = 120 * time.Second
)

var errHostedMcpNotFound = errors.New("managed MCP resource not found")

// HostedMcpRuntime は同じ API のテナントの二つの入口で共有する転送予算を持つ。
type HostedMcpRuntime struct {
	client  *http.Client
	mu      sync.Mutex
	total   int
	tenants map[string]int
}

func NewHostedMcpRuntime(client *http.Client) *HostedMcpRuntime {
	if client == nil {
		client = safehttp.NewClient(safehttp.Config{DialTimeout: 5 * time.Second, TLSTimeout: 5 * time.Second, RequestTimeout: hostedMcpTimeout, ValidateURL: domain.ValidateMcpUpstreamURL})
	}
	cloned := *client
	cloned.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	cloned.Timeout = hostedMcpTimeout
	return &HostedMcpRuntime{client: &cloned, tenants: make(map[string]int)}
}

func (r *HostedMcpRuntime) acquire(tenant string) (func(), int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.total >= 64 {
		return nil, http.StatusServiceUnavailable
	}
	if r.tenants[tenant] >= 8 {
		return nil, http.StatusTooManyRequests
	}
	r.total++
	r.tenants[tenant]++
	return func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.total--
		r.tenants[tenant]--
		if r.tenants[tenant] == 0 {
			delete(r.tenants, tenant)
		}
	}, 0
}

func (d Deps) hostedMcpServer(c *echo.Context) (*domain.McpResourceServer, error) {
	if d.McpResourceServerRepo == nil {
		return nil, tokenusecases.ErrResourceAccessNotConfigured
	}
	server, err := d.McpResourceServerRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), c.Param("server_id"))
	if err != nil {
		return nil, err
	}
	if server == nil || !server.IsActive() || server.Hosting == nil {
		return nil, errHostedMcpNotFound
	}
	return server, nil
}

func (d Deps) handleHostedMcpMetadata(c *echo.Context) error {
	server, err := d.hostedMcpServer(c)
	if errors.Is(err, errHostedMcpNotFound) {
		return hostedMcpError(c, 404, nil, "not found", nil)
	}
	if err != nil {
		return hostedMcpError(c, 503, nil, "metadata is unavailable", nil)
	}
	if server == nil {
		return hostedMcpError(c, 404, nil, "not found", nil)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"resource": server.Resource, "authorization_servers": []string{support.RequestIssuer(c, d.Issuer)},
		"scopes_supported": server.Scopes, "bearer_methods_supported": []string{"header"}, "dpop_signing_alg_values_supported": []string{"PS256", "ES256"},
	})
}

// RegisterHostedMcpPathMetadata は RFC 9728 の origin 直下の経路だけを追加する。
func RegisterHostedMcpPathMetadata(g *echo.Group, d Deps) {
	g.GET("/mcp/:server_id", d.handleHostedMcpMetadata)
}

func (d Deps) handleHostedMcp(c *echo.Context) error {
	server, err := d.hostedMcpServer(c)
	if errors.Is(err, errHostedMcpNotFound) {
		return hostedMcpError(c, 404, nil, "not found", nil)
	}
	if err != nil {
		return hostedMcpError(c, 503, nil, "gateway is unavailable", nil)
	}
	if server == nil {
		return hostedMcpError(c, 404, nil, "not found", nil)
	}
	request := c.Request()
	if request.Method != http.MethodPost {
		c.Response().Header().Set("Allow", "POST")
		return hostedMcpError(c, 405, nil, "POST is required", nil)
	}
	if request.URL.RawQuery != "" || request.URL.ForceQuery {
		return hostedMcpError(c, 400, nil, "query parameters are not supported", nil)
	}
	if origins := request.Header.Values("Origin"); len(origins) > 1 || (len(origins) == 1 && !slices.Contains(server.Hosting.AllowedOrigins, origins[0])) {
		return hostedMcpError(c, 403, nil, "origin is not allowed", nil)
	}
	content, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	encodings := request.Header.Values("Content-Encoding")
	if err != nil || content != "application/json" || len(encodings) > 1 || (len(encodings) == 1 && !strings.EqualFold(encodings[0], "identity")) {
		return hostedMcpError(c, 415, nil, "an uncompressed JSON request is required", nil)
	}
	if d.HostedMcp == nil || d.RateLimiter == nil || d.Emit == nil {
		return hostedMcpError(c, 503, nil, "gateway is unavailable", nil)
	}
	release, status := d.HostedMcp.acquire(server.TenantID)
	if status != 0 {
		c.Response().Header().Set("Retry-After", "1")
		return hostedMcpError(c, status, nil, "gateway capacity exceeded", nil)
	}
	defer release()
	budget, err := d.RateLimiter.AllowWithin(request.Context(), "managed_mcp", fmt.Sprintf("%q:%q", server.TenantID, server.ID), rlports.RateLimitPolicyConfig{MaxRequests: 60, WindowSeconds: 60}, time.Now().UTC())
	if err != nil {
		return hostedMcpError(c, 503, nil, "gateway is unavailable", nil)
	}
	if !budget.Allowed {
		c.Response().Header().Set("Retry-After", fmt.Sprint(budget.RetryAfterSeconds))
		return hostedMcpError(c, 429, nil, "request budget exceeded", nil)
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Response(), request.Body, hostedMcpRequestLimit))
	if err != nil {
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			return hostedMcpError(c, 413, nil, "request body too large", nil)
		}
		return hostedMcpError(c, 400, nil, "request body could not be read", nil)
	}
	message, violation := mcp.Inspect(request.Header.Values, body)
	if violation != nil {
		return support.NoStoreJSON(c, 400, map[string]any{"jsonrpc": "2.0", "id": violation.ID, "error": map[string]any{"code": violation.Code, "message": violation.Message}})
	}
	token, ok := hostedMcpToken(request.Header)
	proofs := request.Header.Values("DPoP")
	if !ok || len(proofs) > 1 || (len(proofs) == 1 && strings.Contains(proofs[0], ",")) {
		return d.hostedMcpChallenge(c, server, message.ID)
	}
	parameters, err := json.Marshal(message.Arguments)
	if err != nil {
		return hostedMcpError(c, 400, message.ID, "invalid operation arguments", nil)
	}
	if message.Arguments == nil {
		parameters = []byte("{}")
	}
	if len(parameters) > maxOperationParametersLength {
		return hostedMcpError(c, 403, message.ID, "operation parameters too large", nil)
	}
	in := tokenusecases.ResourceAccessRequest{TenantID: server.TenantID, ClientID: server.ClientID, Token: token, DPoPProof: request.Header.Get("DPoP"), HTM: http.MethodPost, HTU: server.Resource, OperationKey: rand.Text(), OperationParameters: string(parameters), RequestID: logging.RequestIDFromContext(request.Context())}
	in.Attributes = "null"
	// 属性が大きすぎる場合はオブジェクトを宣言せず、条件を持つ規則を既存評価の
	// ConditionIndeterminate で拒否する。条件のない操作の本文は縮めない。
	if len(parameters) <= maxAttributesLength {
		in.Attributes = string(parameters)
	}
	if message.Method == mcp.MethodToolsCall {
		in.ResourceType, in.Relation, in.ResourceID = "idmagic_mcp_operation", "invoke", message.ToolName
	} else {
		if !slices.Contains([]string{"server/discover", "initialize", "notifications/initialized", "ping", "tools/list"}, message.Method) {
			return hostedMcpError(c, 404, message.ID, "method not found", nil)
		}
		in.ResourceType, in.Relation, in.ResourceID = "idmagic_mcp_method", "access", message.Method
	}
	if server.Hosting.Authentication == "token_exchange" {
		in.TokenExchangeResource = server.Hosting.UpstreamURL
	}
	ctx, cancel := context.WithTimeout(request.Context(), hostedMcpTimeout)
	defer cancel()
	decision, err := tokenusecases.EvaluateResourceAccess(ctx, tokenusecases.ResourceAccessDeps{Servers: d.McpResourceServerRepo, Tokens: tokenusecases.RevocationCheckingIntrospector{Deps: d.introspectDeps()}, Clients: d.ClientRepo, Relations: d.RelationChecker, Replay: d.DpopReplayStore, Approvals: d.ApprovalRequestStore, Limiter: d.RateLimiter, Grants: d.TokenExchangeGrantStore, Emit: d.Emit}, in, time.Now().UTC())
	if err != nil {
		return hostedMcpError(c, 503, message.ID, "authorization is unavailable", nil)
	}
	if !decision.Permitted || (decision.ApprovedOperation != nil && decision.ApprovedOperation.Replayed) {
		if slices.Contains(decision.Reasons, domain.ReasonTokenRejected) || slices.Contains(decision.Reasons, domain.ReasonSenderProofInvalid) {
			return d.hostedMcpChallenge(c, server, message.ID)
		}
		var data any
		if len(decision.Remediation) > 0 {
			data = map[string]any{"authorization_remediation": map[string]any{"authorization_details": decision.Remediation}}
		}
		return hostedMcpError(c, 403, message.ID, "forbidden", data)
	}
	headers := http.Header{}
	for _, name := range []string{"Accept", "Content-Type", "Mcp-Protocol-Version", "Mcp-Method", "Mcp-Name", "Mcp-Session-Id"} {
		if value := request.Header.Get(name); value != "" {
			headers.Set(name, value)
		}
	}
	var upstream *http.Response
	if server.Hosting.Authentication == "token_vault" {
		if d.McpForwarder == nil {
			return hostedMcpError(c, 503, message.ID, "credentials are unavailable", nil)
		}
		upstream, err = d.McpForwarder.ForwardMCP(ctx, oauthports.McpForwardRequest{TenantID: server.TenantID, ConnectorID: server.Hosting.ConnectorID, URL: server.Hosting.UpstreamURL, Caller: decision.Caller, Headers: headers, Body: body})
		if errors.Is(err, oauthports.ErrMcpForwardDenied) {
			return hostedMcpError(c, 403, message.ID, "forbidden", nil)
		}
	} else {
		upstream, err = d.openHostedMcpExchange(ctx, server, token, decision.TokenExchangeGrant, headers, body)
	}
	if err != nil || upstream == nil {
		return hostedMcpError(c, 503, message.ID, "upstream is unavailable", nil)
	}
	defer func() { _ = upstream.Body.Close() }()
	return streamHostedMcp(c, upstream, message.ID, hostedMcpResponseLimit)
}

func hostedMcpToken(headers http.Header) (string, bool) {
	values := headers.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	scheme, token, found := strings.Cut(values[0], " ")
	return token, found && token != "" && !strings.ContainsAny(token, ", \t\r\n") && (strings.EqualFold(scheme, "Bearer") || strings.EqualFold(scheme, "DPoP"))
}

func (d Deps) hostedMcpChallenge(c *echo.Context, server *domain.McpResourceServer, id json.RawMessage) error {
	resource, _ := url.Parse(server.Resource)
	metadata := resource.Scheme + "://" + resource.Host + "/.well-known/oauth-protected-resource" + resource.Path
	c.Response().Header().Set("WWW-Authenticate", fmt.Sprintf("Bearer resource_metadata=%q", metadata))
	return hostedMcpError(c, 401, id, "unauthorized", nil)
}

func hostedMcpError(c *echo.Context, status int, id json.RawMessage, message string, data any) error {
	rpcError := map[string]any{"code": -32000, "message": message}
	if data != nil {
		rpcError["data"] = data
	}
	return support.NoStoreJSON(c, status, map[string]any{"jsonrpc": "2.0", "id": id, "error": rpcError})
}

func (d Deps) openHostedMcpExchange(ctx context.Context, server *domain.McpResourceServer, token, grant string, headers http.Header, body []byte) (*http.Response, error) {
	client, err := d.ClientRepo.FindByID(ctx, server.TenantID, server.ClientID)
	if err != nil || client == nil {
		return nil, oauthports.ErrMcpForwardUnavailable
	}
	d, _, err = d.prepareTokenIssuance(ctx, client, "urn:ietf:params:oauth:grant-type:token-exchange")
	if err != nil {
		return nil, err
	}
	key, err := d.mcpGatewayKey(ctx, server.ID)
	if err != nil {
		return nil, err
	}
	jkt, err := tokens_jose.ValidatePublicSigningJWK(key.PublicJWK)
	if err != nil {
		return nil, err
	}
	result, err := tokenusecases.ExchangeToken(ctx, d.tokenExchangeDeps(), tokenusecases.ExchangeTokenInput{ClientID: server.ClientID, SubjectToken: token, SubjectTokenType: "urn:ietf:params:oauth:token-type:access_token", Resource: []string{server.Hosting.UpstreamURL}, ProofJKT: jkt, TokenExchangeGrant: grant}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(result.TokenType, "DPoP") {
		return nil, oauthports.ErrMcpForwardUnavailable
	}
	sum := sha256.Sum256([]byte(result.AccessToken))
	proof, err := tokens_jose.SignJWT(key.PrivateKey, string(key.Alg), map[string]any{"typ": "dpop+jwt", "jwk": key.PublicJWK}, map[string]any{"htm": "POST", "htu": server.Hosting.UpstreamURL, "iat": time.Now().Unix(), "jti": rand.Text(), "ath": base64.RawURLEncoding.EncodeToString(sum[:])})
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, server.Hosting.UpstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.GetBody = nil
	request.Header = headers.Clone()
	request.Header.Set("Authorization", "DPoP "+result.AccessToken)
	request.Header.Set("DPoP", proof)
	return d.HostedMcp.client.Do(request)
}

func streamHostedMcp(c *echo.Context, upstream *http.Response, id json.RawMessage, limit int64) error {
	content, _, err := mime.ParseMediaType(upstream.Header.Get("Content-Type"))
	if upstream.StatusCode != http.StatusOK || err != nil || (content != "application/json" && content != "text/event-stream") {
		return hostedMcpError(c, 502, id, "upstream response is not supported", nil)
	}
	writer := c.Response()
	controller := http.NewResponseController(writer)
	_ = controller.SetWriteDeadline(time.Now().Add(hostedMcpTimeout))
	writer.Header().Set("Content-Type", content)
	writer.Header().Set("Cache-Control", "no-store")
	if session := upstream.Header.Get("Mcp-Session-Id"); session != "" {
		writer.Header().Set("Mcp-Session-Id", session)
	}
	writer.WriteHeader(http.StatusOK)
	buffer := make([]byte, 32<<10)
	remaining := limit
	for {
		n, readErr := upstream.Body.Read(buffer[:min(int64(len(buffer)), remaining+1)])
		if int64(n) > remaining {
			panic(http.ErrAbortHandler)
		}
		if n > 0 {
			if _, err := writer.Write(buffer[:n]); err != nil {
				panic(http.ErrAbortHandler)
			}
			remaining -= int64(n)
			if content == "text/event-stream" {
				if err := controller.Flush(); err != nil {
					panic(http.ErrAbortHandler)
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			panic(http.ErrAbortHandler)
		}
	}
}
