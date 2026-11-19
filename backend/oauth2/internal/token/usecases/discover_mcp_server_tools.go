package usecases

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/protocol/mcp"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// 取得の上限は REQ-OAUTH2-095 と REQ-OAUTH2-096 が定める。
const (
	discoveryMetadataLimit  = 64 << 10
	discoveryPageLimit      = 4 << 20
	discoveryMaxPages       = 10
	discoveryOverallTimeout = 30 * time.Second
	discoveryTokenLifetime  = 60 * time.Second
	discoveryRatePolicy     = "mcp_tool_discovery"
)

var discoveryRate = rlports.RateLimitPolicyConfig{MaxRequests: 6, WindowSeconds: 60}

var (
	// ErrMcpServerNotFound は取得を求めた登録がテナントにないことを表す。
	ErrMcpServerNotFound = errors.New("MCP resource server not found")
	// ErrMcpServerNotDiscoverable は登録が Active のセルフホスト型でないか、resource が取得の宛先にできないことを表す。
	ErrMcpServerNotDiscoverable = errors.New("MCP resource server cannot be discovered")
	// ErrDiscoveryUnavailable は流量制限を実行できないことを表す。外へは接続していない。
	ErrDiscoveryUnavailable = errors.New("MCP tool discovery is unavailable")
)

// DiscoveryRateLimitedError は流量の上限に達したことと、再試行までの秒数である。
type DiscoveryRateLimitedError struct{ RetryAfterSeconds int }

func (e *DiscoveryRateLimitedError) Error() string { return "MCP tool discovery is rate limited" }

// McpToolDiscoveryDeps は取得の作用である。Fetcher は宛先を制限した外向きの通信である。
type McpToolDiscoveryDeps struct {
	Servers  ports.McpResourceServerRepository
	Catalogs ports.McpToolCatalogRepository
	Limiter  rlports.RateLimiter
	Fetcher  ports.McpServerFetcher
	Tokens   ports.TokenIssuer
	Emit     func(spec.DomainEvent)
}

// McpToolDiscoveryRequest は操作した管理者と、テナントの issuer である。
type McpToolDiscoveryRequest struct {
	TenantID, ServerID, ActorUserID, Issuer string
}

// DiscoveredMetadata はメタデータの取得の結果である。値は found のときだけ持つ。
type DiscoveredMetadata struct {
	Status               domain.MetadataStatus
	URL                  string
	AuthorizationServers []string
	ScopesSupported      []string
	IssuerTrusted        bool
}

// DiscoveredTools は一覧の取得の結果である。件数と complete は fetched のときだけ意味を持つ。
type DiscoveredTools struct {
	Status   domain.ToolsStatus
	Count    int
	Complete bool
}

// McpToolDiscoveryResult は取得の結果である。取得した一覧はカタログに保存し、ここには含めない。
type McpToolDiscoveryResult struct {
	Metadata DiscoveredMetadata
	Tools    DiscoveredTools
}

// DiscoverMcpServerTools は、PRM が IdMagic を認可サーバーとして宣言したセルフホスト型の登録から、
// 一覧取得用のトークンで tools/list を取得してカタログに保存する (REQ-OAUTH2-095)。登録は変えない。
func DiscoverMcpServerTools(ctx context.Context, deps McpToolDiscoveryDeps, in McpToolDiscoveryRequest, now time.Time) (*McpToolDiscoveryResult, error) {
	server, err := deps.Servers.FindByID(ctx, in.TenantID, in.ServerID)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, ErrMcpServerNotFound
	}
	if !server.IsActive() || server.Hosting != nil || domain.ValidateMcpUpstreamURL(server.Resource) != nil {
		return nil, ErrMcpServerNotDiscoverable
	}
	if deps.Limiter == nil {
		return nil, ErrDiscoveryUnavailable
	}
	allowed, err := deps.Limiter.AllowWithin(ctx, discoveryRatePolicy, server.ID, discoveryRate, now)
	if err != nil {
		return nil, ErrDiscoveryUnavailable
	}
	if !allowed.Allowed {
		return nil, &DiscoveryRateLimitedError{RetryAfterSeconds: allowed.RetryAfterSeconds}
	}

	ctx, cancel := context.WithTimeout(ctx, discoveryOverallTimeout)
	defer cancel()
	result := &McpToolDiscoveryResult{Tools: DiscoveredTools{Status: domain.ToolsSkipped}}
	metadata := fetchMetadata(ctx, deps.Fetcher, server.Resource, &result.Metadata)
	if metadata != nil {
		result.Metadata.AuthorizationServers = metadata.AuthorizationServers
		result.Metadata.ScopesSupported = metadata.ScopesSupported
		result.Metadata.IssuerTrusted = metadata.Trusts(in.Issuer)
	}
	if metadata != nil && result.Metadata.IssuerTrusted {
		catalog, status, err := fetchTools(ctx, deps, server, metadata.SupportsES256DPoP(), now)
		if err != nil {
			return nil, err
		}
		result.Tools.Status = status
		if catalog != nil {
			if err := deps.Catalogs.Save(ctx, catalog); err != nil {
				return nil, err
			}
			result.Tools.Count, result.Tools.Complete = len(catalog.Tools), catalog.Complete
		}
	}
	if deps.Emit != nil {
		deps.Emit(&domain.McpToolDiscoveryAttempted{
			At: now, TenantID: server.TenantID, ActorUserID: in.ActorUserID, ResourceServerID: server.ID,
			Resource: server.Resource, MetadataStatus: string(result.Metadata.Status), ToolsStatus: string(result.Tools.Status),
		})
	}
	return result, nil
}

// fetchMetadata は path を継いだ URL から順に GET し、404 のときだけ次の URL を試す。
func fetchMetadata(ctx context.Context, fetcher ports.McpServerFetcher, resource string, out *DiscoveredMetadata) *domain.ProtectedResourceMetadata {
	urls, err := domain.ProtectedResourceMetadataURLs(resource)
	if err != nil {
		out.Status = domain.MetadataInvalid
		return nil
	}
	out.Status = domain.MetadataNotFound
	for _, target := range urls {
		out.URL = target
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
		if err != nil {
			out.Status = domain.MetadataInvalid
			return nil
		}
		request.Header.Set("Accept", "application/json")
		status, body, err := fetcher.Do(request, discoveryMetadataLimit)
		switch {
		case errors.Is(err, ports.ErrMcpResponseTooLarge):
			out.Status = domain.MetadataInvalid
			return nil
		case err != nil:
			out.Status = domain.MetadataUnreachable
			return nil
		case status == http.StatusNotFound:
			continue
		case status != http.StatusOK:
			out.Status = domain.MetadataInvalid
			return nil
		}
		metadata, parsed := domain.ParseProtectedResourceMetadata(body, resource)
		out.Status = parsed
		return metadata
	}
	return nil
}

// fetchTools は一覧取得用のトークンで nextCursor をたどる。一ページ目の失敗だけを状態として返し、
// 二ページ目以降の失敗と上限は REQ-MCPGATEWAY-011 と同じく取得できた分を不完全にする。
func fetchTools(ctx context.Context, deps McpToolDiscoveryDeps, server *domain.McpResourceServer, useDPoP bool, now time.Time) (*domain.McpToolCatalog, domain.ToolsStatus, error) {
	presenter, err := newDiscoveryPresenter(ctx, deps.Tokens, server, useDPoP, now)
	if err != nil {
		return nil, "", err
	}
	var listed []mcp.ListedTool
	complete, cursor := false, ""
	for page := range discoveryMaxPages {
		tools, next, status := fetchToolsPage(ctx, deps.Fetcher, presenter, server.Resource, cursor)
		if status != "" {
			if page == 0 {
				return nil, status, nil
			}
			break
		}
		listed = append(listed, tools...)
		if next == "" {
			complete = true
			break
		}
		cursor = next
	}
	tools, withinLimit, err := domain.CatalogToolsFromListed(listed)
	if err != nil {
		return nil, domain.ToolsInvalidResponse, nil //nolint:nilerr // 上限を満たさない一覧は取得の結果の状態であり、操作の失敗ではない (REQ-OAUTH2-095)
	}
	return &domain.McpToolCatalog{
		TenantID: server.TenantID, ResourceServerID: server.ID, Tools: tools, Complete: complete && withinLimit, ReceivedAt: now,
	}, domain.ToolsFetched, nil
}

func fetchToolsPage(ctx context.Context, fetcher ports.McpServerFetcher, presenter *discoveryPresenter, resource, cursor string) ([]mcp.ListedTool, string, domain.ToolsStatus) {
	body, err := mcp.ToolsListRequest(cursor)
	if err != nil {
		return nil, "", domain.ToolsInvalidResponse
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, resource, bytes.NewReader(body))
	if err != nil {
		return nil, "", domain.ToolsInvalidResponse
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Mcp-Protocol-Version", mcp.ProtocolVersion)
	request.Header.Set("Mcp-Method", mcp.MethodToolsList)
	if err := presenter.present(request); err != nil {
		return nil, "", domain.ToolsInvalidResponse
	}
	status, raw, err := fetcher.Do(request, discoveryPageLimit)
	switch {
	case errors.Is(err, ports.ErrMcpResponseTooLarge):
		return nil, "", domain.ToolsInvalidResponse
	case err != nil:
		return nil, "", domain.ToolsUnreachable
	case status == http.StatusUnauthorized:
		return nil, "", domain.ToolsUnauthorized
	case status == http.StatusForbidden:
		return nil, "", domain.ToolsForbidden
	case status != http.StatusOK:
		return nil, "", domain.ToolsInvalidResponse
	}
	tools, next, err := mcp.ParseToolsListPage(raw)
	if err != nil {
		return nil, "", domain.ToolsInvalidResponse
	}
	return tools, next, ""
}

// discoveryPresenter は一覧取得用のトークンと、DPoP を使うなら取得ごとに作った鍵を持つ (REQ-OAUTH2-096)。
// 鍵は取得が終われば捨て、保存しない。
type discoveryPresenter struct {
	token string
	key   *ecdsa.PrivateKey
	jwk   map[string]any
	htu   string
}

func newDiscoveryPresenter(ctx context.Context, tokens ports.TokenIssuer, server *domain.McpResourceServer, useDPoP bool, now time.Time) (*discoveryPresenter, error) {
	presenter := &discoveryPresenter{htu: server.Resource}
	input := ports.AccessTokenInput{
		Client: &domain.OAuth2Client{TenantID: server.TenantID, ClientID: domain.McpToolDiscoveryClientID},
		Sub:    domain.McpToolDiscoveryClientID, Audiences: []string{server.Resource},
		ExpiresAt: now.Add(discoveryTokenLifetime).Unix(), ToolDiscovery: true,
	}
	if useDPoP {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}
		presenter.key = key
		presenter.jwk = ecPublicJWK(&key.PublicKey)
		jkt, err := tokens_jose.ValidatePublicSigningJWK(presenter.jwk)
		if err != nil {
			return nil, err
		}
		input.SenderConstraint = &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: jkt}
	}
	token, _, err := tokens.SignAccessToken(ctx, input)
	if err != nil {
		return nil, err
	}
	presenter.token = token
	return presenter, nil
}

func (p *discoveryPresenter) present(request *http.Request) error {
	if p.key == nil {
		request.Header.Set("Authorization", "Bearer "+p.token)
		return nil
	}
	proof, err := tokens_jose.SignJWT(p.key, "ES256", map[string]any{"typ": "dpop+jwt", "jwk": p.jwk}, map[string]any{
		"htm": http.MethodPost, "htu": p.htu, "iat": time.Now().Unix(), "jti": rand.Text(), "ath": tokens_jose.AccessTokenHash(p.token),
	})
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "DPoP "+p.token)
	request.Header.Set("DPoP", proof)
	return nil
}

func ecPublicJWK(key *ecdsa.PublicKey) map[string]any {
	encoded, _ := key.Bytes() // 非圧縮形式 0x04 || X || Y。P-256 では各 32 バイトである。
	return map[string]any{
		"kty": "EC", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(encoded[1:33]),
		"y": base64.RawURLEncoding.EncodeToString(encoded[33:65]),
	}
}
