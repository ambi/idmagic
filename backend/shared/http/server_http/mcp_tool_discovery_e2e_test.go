package server_http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

const discoveryResource = "https://mcp.example.com/mcp"

// discoveryMcpServer は PRM と二ページの tools/list を返す MCP サーバーの代役である。
// 受け取った Authorization と DPoP を記録する。
type discoveryMcpServer struct {
	mu            sync.Mutex
	metadata      string
	authorization []string
	proofs        []string
	requested     []string
}

func newDiscoveryMcpServer(issuer string) *discoveryMcpServer {
	return &discoveryMcpServer{metadata: `{"resource":"` + discoveryResource + `","authorization_servers":["` + issuer +
		`"],"scopes_supported":["files:read","files:write"],"dpop_signing_alg_values_supported":["ES256"]}`}
}

func (m *discoveryMcpServer) client() *http.Client {
	return &http.Client{Transport: managedMcpTransport(func(r *http.Request) (*http.Response, error) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.requested = append(m.requested, r.Method+" "+r.URL.String())
		recorder := httptest.NewRecorder()
		recorder.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.String() == "https://mcp.example.com/.well-known/oauth-protected-resource/mcp":
			_, _ = recorder.WriteString(m.metadata)
		case r.Method == http.MethodPost && r.URL.String() == discoveryResource:
			m.authorization = append(m.authorization, r.Header.Get("Authorization"))
			m.proofs = append(m.proofs, r.Header.Get("DPoP"))
			raw, _ := io.ReadAll(r.Body)
			var call struct {
				Method string `json:"method"`
				Params struct {
					Cursor string `json:"cursor"`
				} `json:"params"`
			}
			_ = json.Unmarshal(raw, &call)
			if call.Method != "tools/list" || r.Header.Get("Mcp-Method") != "tools/list" {
				recorder.WriteHeader(http.StatusBadRequest)
				break
			}
			if call.Params.Cursor == "" {
				_, _ = recorder.WriteString(`{"jsonrpc":"2.0","id":1,"result":{"nextCursor":"p2","tools":[{"name":"search","title":"Search","inputSchema":{"type":"object","properties":{"query":{}}}}]}}`)
			} else {
				_, _ = recorder.WriteString(`{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"delete_file","description":"Delete a file"}]}}`)
			}
		default:
			recorder.WriteHeader(http.StatusNotFound)
		}
		return recorder.Result(), nil
	})}
}

func (m *discoveryMcpServer) received() ([]string, []string, []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.authorization), slices.Clone(m.proofs), slices.Clone(m.requested)
}

func newDiscoveryFixture(t *testing.T) (*publicSetupPath, *discoveryMcpServer, string) {
	t.Helper()
	server := newDiscoveryMcpServer(stack.Issuer + "/realms/default")
	f := newPublicSetupPath(t, stack.WithAudit(), stack.WithMcpToolDiscoveryHTTPClient(server.client()))
	created := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers", map[string]any{
		"name": "Public files", "resource": discoveryResource, "scopes": []string{"files:read"},
	}, f.admin, http.StatusCreated)
	return f, server, setupString(t, created, "id")
}

type discoveryReplay struct{ seen map[string]bool }

func (r *discoveryReplay) RecordIfNew(_ context.Context, jti string, _ int, _ time.Time) (bool, error) {
	if r.seen[jti] {
		return false, nil
	}
	r.seen[jti] = true
	return true, nil
}

//spec:covers REQ-OAUTH2-095, REQ-OAUTH2-096: PRM が IdMagic を宣言した公開の MCP サーバーへ、登録の資源だけを宛先とする DPoP のトークンで tools/list をたどり、一覧を登録を変えずにカタログとして保存する。
//verify:operation DiscoverAdminMcpResourceServerTools: 200 と no-store で、メタデータのスコープと認可サーバー、一覧の件数と complete、一覧取得用の client_id を返す。
func TestMcpToolDiscoveryFetchesMetadataAndTools(t *testing.T) {
	f, server, id := newDiscoveryFixture(t)
	before := f.request(t, http.MethodGet, "/api/admin/v1/mcp-resource-servers/"+id, nil, f.admin, http.StatusOK)

	result := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-discovery", nil, f.admin, http.StatusOK)
	if result["client_id"] != oauthdomain.McpToolDiscoveryClientID {
		t.Fatalf("client_id=%v", result["client_id"])
	}
	metadata, _ := result["metadata"].(map[string]any)
	if metadata["status"] != "found" || metadata["issuer_trusted"] != true || !reflectStrings(metadata["scopes_supported"], "files:read", "files:write") {
		t.Fatalf("metadata=%v", metadata)
	}
	tools, _ := result["tools"].(map[string]any)
	if tools["status"] != "fetched" || tools["count"] != float64(2) || tools["complete"] != true {
		t.Fatalf("tools=%v", tools)
	}

	authorization, proofs, _ := server.received()
	if len(authorization) != 2 {
		t.Fatalf("tools/list pages=%d; want 2", len(authorization))
	}
	replay := &discoveryReplay{seen: map[string]bool{}}
	for page, header := range authorization {
		token, ok := strings.CutPrefix(header, "DPoP ")
		if !ok {
			t.Fatalf("page %d authorization=%q; want DPoP", page, header)
		}
		claims := unverifiedClaims(t, token)
		if claims["aud"] != discoveryResource || claims["sub"] != oauthdomain.McpToolDiscoveryClientID || claims["client_id"] != oauthdomain.McpToolDiscoveryClientID || claims["scope"] != "" {
			t.Fatalf("page %d claims=%v", page, claims)
		}
		if lifetime := claims["exp"].(float64) - claims["iat"].(float64); lifetime > 61 {
			t.Fatalf("page %d lifetime=%v; want at most 60 seconds", page, lifetime)
		}
		proof, err := tokens_jose.VerifyDPoPForResource(context.Background(), proofs[page], http.MethodPost, discoveryResource, token, replay, time.Now())
		if err != nil {
			t.Fatalf("page %d proof: %v", page, err)
		}
		cnf, _ := claims["cnf"].(map[string]any)
		if cnf["jkt"] != proof.JKT {
			t.Fatalf("page %d cnf=%v proof jkt=%s", page, cnf, proof.JKT)
		}
	}

	catalog := f.request(t, http.MethodGet, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-catalog", nil, f.admin, http.StatusOK)
	var names []string
	for _, tool := range catalog["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	slices.Sort(names)
	if !slices.Equal(names, []string{"delete_file", "search"}) || catalog["complete"] != true || catalog["received_at"] == nil {
		t.Fatalf("catalog=%v", catalog)
	}
	after := f.request(t, http.MethodGet, "/api/admin/v1/mcp-resource-servers/"+id, nil, f.admin, http.StatusOK)
	if after["updated_at"] != before["updated_at"] || !reflectStrings(after["scopes"], "files:read") {
		t.Fatalf("registration changed: before=%v after=%v", before, after)
	}

	var attempted *oauthdomain.McpToolDiscoveryAttempted
	for _, recorded := range f.s.Events.All() {
		if event, ok := recorded.(*oauthdomain.McpToolDiscoveryAttempted); ok {
			attempted = event
		}
	}
	if attempted == nil || attempted.ResourceServerID != id || attempted.MetadataStatus != "found" || attempted.ToolsStatus != "fetched" || attempted.ActorUserID == "" {
		t.Fatalf("event=%+v", attempted)
	}
}

//spec:covers REQ-OAUTH2-095: PRM が IdMagic を認可サーバーとして宣言しないか resource が登録と異なる場合は、トークンを発行せず tools/list を送らず、カタログを作らない。
func TestMcpToolDiscoverySendsNoTokenWithoutTrust(t *testing.T) {
	cases := map[string]struct {
		metadata string
		status   string
	}{
		"other issuer":      {`{"resource":"` + discoveryResource + `","authorization_servers":["https://other.example"]}`, "found"},
		"resource mismatch": {`{"resource":"https://mcp.example.com/","authorization_servers":["` + stack.Issuer + `/realms/default"]}`, "resource_mismatch"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f, server, id := newDiscoveryFixture(t)
			server.metadata = tc.metadata
			result := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-discovery", nil, f.admin, http.StatusOK)
			metadata, _ := result["metadata"].(map[string]any)
			tools, _ := result["tools"].(map[string]any)
			if metadata["status"] != tc.status || tools["status"] != "skipped" {
				t.Fatalf("result=%v", result)
			}
			if tc.status == "resource_mismatch" && metadata["authorization_servers"] != nil {
				t.Fatalf("mismatched metadata exposed its values: %v", metadata)
			}
			if authorization, _, _ := server.received(); len(authorization) != 0 {
				t.Fatalf("tools/list was sent: %v", authorization)
			}
			catalog := f.request(t, http.MethodGet, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-catalog", nil, f.admin, http.StatusOK)
			if catalog["received_at"] != nil {
				t.Fatalf("catalog=%v", catalog)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-096: MCP サーバーが受け取った一覧取得用のトークンは、introspection で active=false になり、トークン交換と管理 API で拒否される。
func TestMcpToolDiscoveryTokenIsRejectedByIdMagic(t *testing.T) {
	f, server, id := newDiscoveryFixture(t)
	f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-discovery", nil, f.admin, http.StatusOK)
	authorization, _, _ := server.received()
	if len(authorization) == 0 {
		t.Fatal("no token reached the MCP server")
	}
	token := strings.TrimPrefix(authorization[0], "DPoP ")

	clientID, key := f.keyClient(t, "files:read", "urn:ietf:params:oauth:grant-type:token-exchange")
	introspected := f.signedForm(t, key, clientID, "/introspect", url.Values{"token": {token}}, http.StatusOK)
	if introspected["active"] != false {
		t.Fatalf("introspection=%v", introspected)
	}
	exchanged := f.signedForm(t, key, clientID, "/token", url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}, "subject_token": {token},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"}, "resource": {discoveryResource},
	}, http.StatusBadRequest)
	if exchanged["error"] != "invalid_grant" || exchanged["access_token"] != nil {
		t.Fatalf("exchange=%v", exchanged)
	}
	f.request(t, http.MethodGet, "/api/admin/v1/mcp-resource-servers", nil, nil, http.StatusUnauthorized, token)
}

//spec:covers REQ-OAUTH2-095: 取得できない登録と流量の超過は外へ接続せずに拒否する。
func TestMcpToolDiscoveryRejectsUnsupportedRegistrationsAndBursts(t *testing.T) {
	f, server, id := newDiscoveryFixture(t)
	f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/missing/tool-discovery", nil, f.admin, http.StatusNotFound)
	disabled := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers", map[string]any{
		"name": "Disabled", "resource": "https://disabled.example.com/mcp", "scopes": []string{"read"}, "state": "Disabled",
	}, f.admin, http.StatusCreated)
	refused := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/"+setupString(t, disabled, "id")+"/tool-discovery", nil, f.admin, http.StatusBadRequest)
	if refused["type"] != "urn:idmagic:error:invalid_resource_server" {
		t.Fatalf("disabled=%v", refused)
	}
	if _, _, requested := server.received(); len(requested) != 0 {
		t.Fatalf("contacted the network for refused registrations: %v", requested)
	}

	for range 6 {
		f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-discovery", nil, f.admin, http.StatusOK)
	}
	_, _, before := server.received()
	limited := f.request(t, http.MethodPost, "/api/admin/v1/mcp-resource-servers/"+id+"/tool-discovery", nil, f.admin, http.StatusTooManyRequests)
	if limited["error"] != "rate_limited" {
		t.Fatalf("limited=%v", limited)
	}
	if _, _, after := server.received(); len(after) != len(before) {
		t.Fatalf("contacted the network after the limit: %v", after[len(before):])
	}
}

func reflectStrings(value any, want ...string) bool {
	items, ok := value.([]any)
	if !ok || len(items) != len(want) {
		return false
	}
	for index, item := range items {
		if item != want[index] {
			return false
		}
	}
	return true
}
