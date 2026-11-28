package server_http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-OAUTH2-099, REQ-MCPGATEWAY-015, REQ-OAUTH2-092: 公開登録と割当から一覧を絞り、既発行トークンでも割当の変更を次の一覧へ反映する。
//verify:operation HostedMcpRequest: 一覧に許可した定義とカーソルだけが現れ、未割当の実行は拒否される。
func TestHostedMcpToolListsFollowCurrentAssignments(t *testing.T) {
	forwarded := 0
	client := &http.Client{Transport: managedMcpTransport(func(r *http.Request) (*http.Response, error) {
		forwarded++
		if r.URL.String() != "https://upstream.example/mcp" || !strings.HasPrefix(r.Header.Get("Authorization"), "DPoP ") {
			t.Fatalf("wrong upstream or credential: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":7,"result":{"resultType":"complete","tools":[{"name":"read_document","inputSchema":{"type":"object"}},{"name":"delete_document"},{"name":"hidden"}],"nextCursor":"page-2","ttlMs":300000,"cacheScope":"public"}}`))}, nil
	})}
	f := newPublicSetupPath(t, stack.WithAccessControl(), stack.WithManagedMcpHTTPClient(client))
	owner := f.user(t, "list-owner")
	agentClient, key := f.keyClient(t, "read", "client_credentials")
	agent := f.agent(t, owner, agentClient, "autonomous")
	evaluator, _ := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	f.request(t, "POST", "/api/admin/v1/mcp-resource-servers", map[string]any{"name": "Upstream", "resource": "https://upstream.example/mcp", "scopes": []string{"read"}}, f.admin, 201)
	rule := func(name string) map[string]any {
		return map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{"read"}, "token_exchange_resources": []string{"https://upstream.example/mcp"}, "tool": map[string]any{"name": name, "display_name": name}}
	}
	server := f.request(t, "POST", "/api/admin/v1/mcp-resource-servers", map[string]any{"name": "Listed documents", "scopes": []string{"read"}, "client_id": evaluator, "hosting": map[string]any{"upstream_url": "https://upstream.example/mcp", "authentication": "token_exchange"}, "access_rules": []any{rule("read_document"), rule("delete_document")}}, f.admin, 201)
	resource := setupString(t, server, "resource")
	rules := server["access_rules"].([]any)
	tuple := func(index int) map[string]any {
		tool := rules[index].(map[string]any)["tool"].(map[string]any)
		return map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": tool["assignment_resource_id"], "relation": "invoke", "subject_type": "agent", "subject_id": agent}
	}
	f.request(t, "POST", "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []any{tuple(0)}}, f.admin, 200)
	issued := f.signedForm(t, key, agentClient, "/token", url.Values{"grant_type": {"client_credentials"}, "resource": {resource}}, 200)
	token := setupString(t, issued, "access_token")
	list := func(want string) {
		t.Helper()
		request := httptest.NewRequest("POST", resource, strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
		request.Header.Set("Mcp-Method", "tools/list")
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(recorder, request)
		var body struct {
			Result struct {
				Tools      []struct{ Name string }
				NextCursor string
				CacheScope string
				TTL        int `json:"ttlMs"`
			}
		}
		if recorder.Code != 200 || json.Unmarshal(recorder.Body.Bytes(), &body) != nil || len(body.Result.Tools) != 1 || body.Result.Tools[0].Name != want || body.Result.NextCursor != "page-2" || body.Result.CacheScope != "private" || body.Result.TTL != 0 || recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("unfiltered list: %d %s %v", recorder.Code, recorder.Body.String(), recorder.Header())
		}
	}
	list("read_document")
	f.request(t, "POST", "/api/admin/v1/authorization/relation-tuples", map[string]any{"deletes": []any{tuple(0)}, "writes": []any{tuple(1)}}, f.admin, 200)
	list("delete_document")
	managedMcpCall(t, f, resource, token, "read_document", 403)
	if forwarded != 2 {
		t.Fatalf("unexpected upstream requests: %d", forwarded)
	}
}

//spec:covers REQ-OAUTH2-099, REQ-MCPGATEWAY-015: 独立配布の Gateway と実際の MCP SDK が公開された評価応答を使って一覧を絞り、割当取消後は取得を拒否する。
//verify:operation EvaluateResourceAccess: tools/list の許可応答の名前と、不許可応答を独立した Gateway と SDK の結果で観測する。
func TestCatalogGatewayToolListUsesEvaluationNames(t *testing.T) {
	f := newMcpGatewayFixture(t, true)
	server, err := f.s.McpResourceServers.FindByResource(context.Background(), tenancydomain.DefaultTenantID, agentRevocationResource)
	if err != nil || server == nil {
		t.Fatalf("server=%v err=%v", server, err)
	}
	server.AccessRules = []oauthdomain.ResourceAccessRule{{ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"tools:read"}, Tool: &oauthdomain.ResourceAccessTool{ID: "read", Name: "read_document", DisplayName: "閲覧"}}}
	if err := f.s.McpResourceServers.Save(t.Context(), server); err != nil {
		t.Fatal(err)
	}
	tuple := map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": server.ID + "/read", "relation": "invoke", "subject_type": "agent", "subject_id": "A1"}
	f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []any{tuple}})
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	session := f.connect(t, token)
	result, err := session.ListTools(t.Context(), nil)
	if err != nil || len(result.Tools) != 1 || result.Tools[0].Name != "read_document" {
		t.Fatalf("list=%+v err=%v", result, err)
	}
	f.adminSession(t, "/api/admin/v1/authorization/relation-tuples", map[string]any{"deletes": []any{tuple}})
	if _, err := session.ListTools(t.Context(), nil); err == nil {
		t.Fatal("listed tools after assignment revocation")
	}
	if f.effects.Load() != 0 || f.deleted.Load() != 0 {
		t.Fatal("tool discovery executed a business operation")
	}
}
