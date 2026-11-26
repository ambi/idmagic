package usecases

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	rlmemory "github.com/ambi/idmagic/backend/shared/ratelimit/db_memory"
	"github.com/ambi/idmagic/backend/shared/testing_mcptwin"
)

// redirectFetcher は登録の resource の origin への要求を、テストの上流へ送り直す。
// 宛先の制限は discovery_http の検査が確かめるので、ここでは本文の上限だけを本番と同じく扱う。
type redirectFetcher struct {
	upstream *url.URL
	client   *http.Client
}

func (f redirectFetcher) Do(request *http.Request, limit int64) (int, []byte, error) {
	redirected := request.Clone(request.Context())
	redirected.URL.Scheme, redirected.URL.Host, redirected.Host = f.upstream.Scheme, f.upstream.Host, ""
	response, err := f.client.Do(redirected)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return 0, nil, err
	}
	if int64(len(body)) > limit {
		return 0, nil, oauthports.ErrMcpResponseTooLarge
	}
	return response.StatusCode, body, nil
}

type discoveryTokens struct{}

func (discoveryTokens) SignAccessToken(context.Context, oauthports.AccessTokenInput) (string, string, error) {
	return "discovery-token", "jti", nil
}

func (discoveryTokens) SignIDToken(context.Context, oauthports.IDTokenInput) (string, error) {
	return "", nil
}
func (discoveryTokens) AccessTokenTTLSeconds() int { return 60 }
func (discoveryTokens) IDTokenTTLSeconds() int     { return 60 }

type discoveryFixture struct {
	deps     McpToolDiscoveryDeps
	catalogs *db_memory.McpToolCatalogRepository
	in       McpToolDiscoveryRequest
}

// newDiscoveryFixture は resource を持つ Active のセルフホスト型の登録を置き、upstream へ接続する取得の作用を組む。
func newDiscoveryFixture(t *testing.T, resource, issuer string, upstream *httptest.Server) discoveryFixture {
	t.Helper()
	servers := db_memory.NewMcpResourceServerRepository()
	servers.Seed(&oauthdomain.McpResourceServer{
		TenantID: "tenant-a", ID: "server-a", Resource: resource, Name: "MCP", Scopes: []string{}, State: oauthdomain.McpResourceServerActive,
	})
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	catalogs := db_memory.NewMcpToolCatalogRepository()
	return discoveryFixture{
		deps: McpToolDiscoveryDeps{
			Servers: servers, Catalogs: catalogs, Limiter: rlmemory.NewRateLimiter(nil),
			Fetcher: redirectFetcher{upstream: target, client: upstream.Client()}, Tokens: discoveryTokens{},
		},
		catalogs: catalogs,
		in:       McpToolDiscoveryRequest{TenantID: "tenant-a", ServerID: "server-a", ActorUserID: "admin", Issuer: issuer},
	}
}

func (f discoveryFixture) discover(t *testing.T) (*McpToolDiscoveryResult, *oauthdomain.McpToolCatalog) {
	t.Helper()
	result, err := DiscoverMcpServerTools(context.Background(), f.deps, f.in, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := f.catalogs.Find(context.Background(), "tenant-a", "server-a")
	if err != nil {
		t.Fatal(err)
	}
	return result, catalog
}

// RFC 9728 §3.2 の例の資源と、その authorization_servers の一つ目である。
const (
	recordedResource = "https://resource.example.com"
	recordedIssuer   = "https://as1.example.com"
)

// MCP 2026-07-28 の server/tools の例のツールを、カタログの項目に読んだ値である。
var recordedCatalog = []oauthdomain.CatalogTool{
	{Name: "get_weather", Title: "Weather Information Provider", Description: "Get current weather information for a location", Arguments: []string{"location"}},
	{Name: "get_current_time", Description: "Returns the current server time", Arguments: []string{}},
}

func recordedDiscovery(t *testing.T, mcpExchanges ...string) *httptest.Server {
	t.Helper()
	exchanges := []testing_mcptwin.Exchange{testing_mcptwin.RFC9728Recording(t).Exchange(t, "protected-resource-metadata")}
	exchanges = append(exchanges, testing_mcptwin.MCP20260728Recording(t).Select(t, mcpExchanges...).Exchanges...)
	return testing_mcptwin.NewReplayServer(t, exchanges...)
}

//spec:covers REQ-OAUTH2-095: RFC 9728 の例の PRM を読み、IdMagic を宣言した資源へ MCP の例の tools/list を二ページたどってカタログに保存する。
func TestDiscoverMcpServerTools_ReadsRecordedMetadataAndTools(t *testing.T) {
	f := newDiscoveryFixture(t, recordedResource, recordedIssuer, recordedDiscovery(t, "tools-list-first-page", "tools-list-last-page"))

	result, catalog := f.discover(t)
	wantMetadata := DiscoveredMetadata{
		Status: oauthdomain.MetadataFound, URL: recordedResource + "/.well-known/oauth-protected-resource",
		AuthorizationServers: []string{"https://as1.example.com", "https://as2.example.net"}, ScopesSupported: []string{"profile", "email", "phone"},
		IssuerTrusted: true,
	}
	if !reflect.DeepEqual(result.Metadata, wantMetadata) || result.Tools != (DiscoveredTools{Status: oauthdomain.ToolsFetched, Count: 2, Complete: true}) {
		t.Fatalf("DiscoverMcpServerTools() = %+v, want metadata %+v and 2 complete tools", result, wantMetadata)
	}
	if catalog == nil || !reflect.DeepEqual(catalog.Tools, recordedCatalog) || !catalog.Complete {
		t.Errorf("catalog = %+v, want %+v complete", catalog, recordedCatalog)
	}
}

//spec:covers REQ-OAUTH2-095: 空文字列の nextCursor を終わりと読まず、cursor に空文字列を載せて次のページを取り、両ページをカタログに保存する。
func TestDiscoverMcpServerTools_FollowsEmptyNextCursor(t *testing.T) {
	f := newDiscoveryFixture(t, recordedResource, recordedIssuer, recordedDiscovery(t, "tools-list-empty-cursor", "tools-list-after-empty-cursor"))

	result, catalog := f.discover(t)
	if result.Tools != (DiscoveredTools{Status: oauthdomain.ToolsFetched, Count: 2, Complete: true}) {
		t.Fatalf("tools = %+v, want 2 complete tools", result.Tools)
	}
	if catalog == nil || !reflect.DeepEqual(catalog.Tools, recordedCatalog) {
		t.Errorf("catalog = %+v, want %+v", catalog, recordedCatalog)
	}
}

//spec:covers REQ-OAUTH2-095: MCP の例の 401 と 403 の一ページ目は unauthorized と forbidden にし、カタログを変えない。
func TestDiscoverMcpServerTools_ClassifiesRecordedRefusals(t *testing.T) {
	for exchange, want := range map[string]oauthdomain.ToolsStatus{
		"tools-list-unauthorized":       oauthdomain.ToolsUnauthorized,
		"tools-list-insufficient-scope": oauthdomain.ToolsForbidden,
	} {
		t.Run(exchange, func(t *testing.T) {
			f := newDiscoveryFixture(t, recordedResource, recordedIssuer, recordedDiscovery(t, exchange))
			result, catalog := f.discover(t)
			if result.Tools.Status != want || catalog != nil {
				t.Errorf("tools = %+v and catalog %+v, want %s and no catalog", result.Tools, catalog, want)
			}
		})
	}
}

// RFC 9728 の例の authorization_servers にテナントの issuer がなければ、トークンを出さず tools/list を送らない。
//
//spec:covers REQ-OAUTH2-095: PRM がテナントの issuer を宣言しなければ一覧を skipped にする。
func TestDiscoverMcpServerTools_SkipsToolsWhenTheRecordedMetadataDoesNotTrustTheIssuer(t *testing.T) {
	f := newDiscoveryFixture(t, recordedResource, "https://idmagic.example.com/t1", recordedDiscovery(t))
	result, _ := f.discover(t)
	if result.Metadata.Status != oauthdomain.MetadataFound || result.Metadata.IssuerTrusted || result.Tools.Status != oauthdomain.ToolsSkipped {
		t.Errorf("DiscoverMcpServerTools() = %+v, want found, untrusted and skipped", result)
	}
}

const twinResource = "https://mcp.example.com/mcp"

func discoveryTwin(names ...string) *testing_mcptwin.Twin {
	tw := testing_mcptwin.NewTwin()
	tw.Resource, tw.AuthorizationServers, tw.ScopesSupported = twinResource, []string{recordedIssuer}, []string{"files:read"}
	tw.Tokens = map[string][]string{"discovery-token": nil}
	for _, name := range names {
		tw.AddTool(map[string]any{"name": name, "inputSchema": map[string]any{"type": "object"}})
	}
	return tw
}

func twinDiscovery(t *testing.T, tw *testing_mcptwin.Twin) discoveryFixture {
	t.Helper()
	upstream := httptest.NewServer(tw)
	t.Cleanup(upstream.Close)
	return newDiscoveryFixture(t, twinResource, recordedIssuer, upstream)
}

//spec:covers REQ-OAUTH2-095: 模倣の上流の path を継いだ PRM を読み、空文字列を含む不透明なカーソルで一覧をたどる。
func TestDiscoverMcpServerTools_FollowsTwinPagination(t *testing.T) {
	tw := discoveryTwin("a", "b", "c")
	tw.PageSize, tw.Cursors = 1, []string{"", "opaque=="}
	result, catalog := twinDiscovery(t, tw).discover(t)
	if result.Metadata.URL != "https://mcp.example.com/.well-known/oauth-protected-resource/mcp" || result.Tools != (DiscoveredTools{Status: oauthdomain.ToolsFetched, Count: 3, Complete: true}) {
		t.Fatalf("DiscoverMcpServerTools() = %+v, want path-inherited metadata and 3 complete tools", result)
	}
	if catalog == nil || len(catalog.Tools) != 3 || catalog.Tools[2].Name != "c" {
		t.Errorf("catalog = %+v, want a, b, c", catalog)
	}
}

//spec:covers REQ-OAUTH2-095: 二ページ目の 5xx と不正な本文は取得できた分を不完全とし、一ページ目の不正な本文と未対応の版は invalid_response にする。
func TestDiscoverMcpServerTools_HandlesTwinFaults(t *testing.T) {
	for _, tc := range []struct {
		name     string
		setup    func(tw *testing_mcptwin.Twin)
		status   oauthdomain.ToolsStatus
		count    int
		complete bool
	}{
		{"二ページ目が 503", func(tw *testing_mcptwin.Twin) {
			tw.InjectFaults(testing_mcptwin.Fault{}, testing_mcptwin.Fault{}, testing_mcptwin.Fault{Status: 503})
		}, oauthdomain.ToolsFetched, 1, false},
		{"二ページ目が不正な本文", func(tw *testing_mcptwin.Twin) {
			tw.InjectFaults(testing_mcptwin.Fault{}, testing_mcptwin.Fault{}, testing_mcptwin.Fault{Status: 200, ContentType: "application/json", Body: "{"})
		}, oauthdomain.ToolsFetched, 1, false},
		{"一ページ目が不正な本文", func(tw *testing_mcptwin.Twin) {
			tw.InjectFaults(testing_mcptwin.Fault{}, testing_mcptwin.Fault{Status: 200, ContentType: "application/json", Body: "<html>"})
		}, oauthdomain.ToolsInvalidResponse, 0, false},
		{"上流が 2026-07-28 を受け付けない", func(tw *testing_mcptwin.Twin) {
			tw.Supported = []string{"2025-11-25"}
		}, oauthdomain.ToolsInvalidResponse, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tw := discoveryTwin("a", "b")
			tw.PageSize = 1
			tc.setup(tw)
			result, _ := twinDiscovery(t, tw).discover(t)
			if result.Tools != (DiscoveredTools{Status: tc.status, Count: tc.count, Complete: tc.complete}) {
				t.Errorf("tools = %+v, want %s with %d tools and complete=%t", result.Tools, tc.status, tc.count, tc.complete)
			}
		})
	}
}
