package server_http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/security/tokens_jose"
)

type managedMcpTransport func(*http.Request) (*http.Response, error)

func (f managedMcpTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func managedMcpCall(t *testing.T, f *publicSetupPath, resource, token, name string, want int, arguments ...string) *httptest.ResponseRecorder {
	t.Helper()
	args := "{}"
	if len(arguments) > 0 {
		args = arguments[0]
	}
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + name + `","arguments":` + args + `,"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
	request := httptest.NewRequest(http.MethodPost, resource, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", name)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Cookie", "private=session")
	request.Header.Set("X-Request-ID", "caller-chosen")
	request.Header.Set("Forwarded", "for=internal")
	recorder := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(recorder, request)
	if recorder.Code != want {
		t.Fatalf("managed MCP %s: status=%d, body=%s; want %d", resource, recorder.Code, recorder.Body.String(), want)
	}
	return recorder
}

//spec:covers REQ-TOKENVAULT-016, REQ-OAUTH2-092: 公開した本人の連携と代行だけで、本人の接続を選び、別本人の接続へ退避せず、401 の MCP 操作を再送しない。
func TestHostedMcpUsesOnlyTheDelegatingUsersVaultConnection(t *testing.T) {
	var sent []string
	refreshes := 0
	status := http.StatusOK
	upstream := &http.Client{Transport: managedMcpTransport(func(r *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		recorder.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case vaultTokenURL:
			_ = r.ParseForm()
			if r.PostForm.Get("grant_type") == "refresh_token" {
				refreshes++
			}
			_, _ = fmt.Fprintf(recorder, `{"access_token":%q,"refresh_token":"refresh","token_type":"Bearer","expires_in":3600,"scope":"read"}`, "stored-"+r.PostForm.Get("code"))
		case vaultRevokeURL:
			recorder.WriteHeader(200)
		default:
			sent = append(sent, r.Header.Get("Authorization"))
			recorder.WriteHeader(status)
			_, _ = recorder.WriteString(`{"jsonrpc":"2.0","id":1,"result":{}}`)
		}
		return recorder.Result(), nil
	})}
	f := newPublicSetupPath(t, stack.WithAccessControl(), stack.WithTokenVault(upstream), stack.WithAudit())
	alice := f.user(t, "managed-alice")
	f.user(t, "managed-bob")
	agentClient, agentKey := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	agent := f.agent(t, alice, agentClient, "autonomous")
	connector := f.request(t, "POST", "/api/admin/v1/token-vault/connectors", map[string]any{
		"name": "Managed upstream", "authorization_endpoint": vaultAuthorizeURL, "token_endpoint": vaultTokenURL, "revocation_endpoint": vaultRevokeURL, "client_id": "vault-client", "client_secret": "vault-secret", "allowed_scopes": []string{"read"},
		"rules": []any{map[string]any{"id": "mcp", "method": "POST", "origin": vaultAPIOrigin, "path_prefix": "/mcp", "required_scopes": []string{"read"}}},
	}, f.admin, 201)
	connectorID := setupString(t, connector, "id")
	f.request(t, "POST", "/api/admin/v1/authorization-detail-types", map[string]any{"type": "token_vault_connection", "display_template": "{identifier}: {actions}", "schema": map[string]any{"rules": []any{
		map[string]any{"name": "identifier", "semantics": "exact", "required": true},
		map[string]any{"name": "actions", "semantics": "set", "required": true},
		map[string]any{"name": "locations", "semantics": "set", "required": true},
	}}}, f.admin, 201)
	evaluator, _ := f.keyClient(t, "read", "client_credentials")
	server := f.request(t, "POST", "/api/admin/v1/mcp-resource-servers", map[string]any{"name": "Managed vault", "scopes": []string{"read"}, "client_id": evaluator, "hosting": map[string]any{"upstream_url": vaultAPIOrigin + "/mcp", "authentication": "token_vault", "connector_id": connectorID}, "access_rules": []any{map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{"read"}, "tool": map[string]any{"name": "read_document", "display_name": "Read documents"}}}}, f.admin, 201)
	resource := setupString(t, server, "resource")
	tool := server["access_rules"].([]any)[0].(map[string]any)["tool"].(map[string]any)
	for _, subject := range []struct{ kind, id string }{{"user", alice}, {"agent", agent}} {
		f.request(t, "POST", "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []any{map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": tool["assignment_resource_id"], "relation": "invoke", "subject_type": subject.kind, "subject_id": subject.id}}}, f.admin, 200)
	}
	browserClient, browserKey := f.keyClient(t, "openid read", "authorization_code")
	details, err := json.Marshal([]any{map[string]any{"type": "token_vault_connection", "identifier": connectorID, "actions": []string{"POST"}, "locations": []string{vaultAPIOrigin}}})
	if err != nil {
		t.Fatal(err)
	}
	code, aliceCookie := f.login(t, "managed-alice", browserClient, "openid read", map[string]string{"authorization_details": string(details)})
	subject := f.signedForm(t, browserKey, browserClient, "/token", url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {setupRedirect}, "code_verifier": {setupVerifier}}, 200)
	_, bobCookie := f.login(t, "managed-bob", browserClient, "openid read")
	link := func(cookie *http.Cookie, code string) {
		t.Helper()
		result := f.request(t, "POST", "/api/account/v1/token-vault/connections/"+connectorID+"/authorize", map[string]any{}, cookie, 200)
		authorization, err := url.Parse(setupString(t, result, "authorization_url"))
		if err != nil {
			t.Fatal(err)
		}
		query := url.Values{"state": {authorization.Query().Get("state")}, "code": {code}}
		request := httptest.NewRequest("GET", stack.Issuer+"/realms/default/api/token-vault/v1/callback/"+connectorID+"?"+query.Encode(), http.NoBody)
		request.AddCookie(cookie)
		recorder := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(recorder, request)
		if recorder.Code != 302 || !strings.Contains(recorder.Header().Get("Location"), "result=linked") {
			t.Fatalf("link %s: %d %s %s", code, recorder.Code, recorder.Header().Get("Location"), recorder.Body.String())
		}
	}
	link(bobCookie, "bob")
	f.request(t, "POST", "/api/account/v1/agent-delegations", map[string]any{"agent_id": agent, "scopes": []string{"read"}}, aliceCookie, 201)
	issued := f.signedForm(t, agentKey, agentClient, "/token", url.Values{"grant_type": {"urn:ietf:params:oauth:grant-type:token-exchange"}, "subject_token_type": {"urn:ietf:params:oauth:token-type:access_token"}, "subject_token": {setupString(t, subject, "access_token")}, "resource": {resource}, "scope": {"read"}}, 200)
	token := setupString(t, issued, "access_token")
	managedMcpCall(t, f, resource, token, "read_document", 403)
	if len(sent) != 0 {
		t.Fatal("used another user's connection")
	}
	f.s.Events.AssertEmitted(t, "TokenVaultExecutionDenied")
	link(aliceCookie, "alice")
	managedMcpCall(t, f, resource, token, "read_document", 200)
	if len(sent) != 1 || sent[0] != "Bearer stored-alice" {
		t.Fatalf("wrong user's credentials: %v", sent)
	}
	f.s.Events.AssertEmitted(t, "TokenVaultExecutionPerformed")
	status = 401
	managedMcpCall(t, f, resource, token, "read_document", 502)
	if len(sent) != 2 || refreshes != 0 {
		t.Fatalf("MCP replay after 401: sends=%v refreshes=%d", sent, refreshes)
	}
	f.request(t, "DELETE", "/api/account/v1/token-vault/connections/"+connectorID, nil, aliceCookie, 204)
	managedMcpCall(t, f, resource, token, "read_document", 403)
	if len(sent) != 2 {
		t.Fatal("disconnected user's request was sent")
	}
}

//spec:covers REQ-OAUTH2-091, REQ-OAUTH2-092, REQ-SIGNINGKEYS-014: 公開登録から割当と取消までを通し、同じ入力資格情報を上流へ渡さず宛先と鍵を分ける。
//verify:operation HostedMcpRequest: 登録と正式なトークンから転送と拒否を観測する。
//verify:operation HostedMcpMetadata: issuer 配下の公開資源のメタデータを観測する。
//verify:operation HostedMcpPathMetadata: origin 直下の経路形式のメタデータを観測する。
func TestHostedMcpPublicSetupEnforcesAssignments(t *testing.T) {
	var forwarded int
	var downstreamToken, downstreamProof string
	expectedArguments := "{}"
	var hold atomic.Bool
	opened := make(chan struct{}, 8)
	finish := make(chan struct{})
	finishStreams := sync.OnceFunc(func() { close(finish) })
	defer finishStreams()
	client := &http.Client{Transport: managedMcpTransport(func(r *http.Request) (*http.Response, error) {
		if hold.Load() {
			reader, writer := io.Pipe()
			go func() {
				_, _ = io.WriteString(writer, "data: waiting\n\n")
				select {
				case <-finish:
				case <-r.Context().Done():
				}
				_ = writer.Close()
			}()
			opened <- struct{}{}
			return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: reader}, nil
		}
		forwarded++
		if r.URL.String() != "https://upstream.example/mcp" || r.Method != http.MethodPost {
			t.Fatalf("unexpected upstream %s %s", r.Method, r.URL)
		}
		for _, name := range []string{"Cookie", "Forwarded", "X-Request-ID"} {
			if r.Header.Get(name) != "" {
				t.Fatalf("forwarded private header %s", name)
			}
		}
		downstreamToken = strings.TrimPrefix(r.Header.Get("Authorization"), "DPoP ")
		downstreamProof = r.Header.Get("DPoP")
		if r.GetBody != nil {
			t.Fatal("POST can be replayed by the transport")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"arguments":`+expectedArguments) {
			t.Fatalf("rewritten body: %s", body)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}, "Set-Cookie": {"upstream=secret"}, "Authorization": {"Bearer upstream-secret"}}, Body: io.NopCloser(strings.NewReader(`{"jsonrpc":"2.0","id":1,"result":{}}`))}, nil
	})}
	f := newPublicSetupPath(t, stack.WithAccessControl(), stack.WithManagedMcpHTTPClient(client), stack.WithAdmissionBudget(support.AdmissionBudget{Enabled: true, MaxConcurrent: 9, ManagementLimit: 8, ManagementBulkLimit: 4}))
	owner := f.user(t, "managed-owner")
	agentClient, key := f.keyClient(t, "read", "client_credentials")
	agent := f.agent(t, owner, agentClient, "autonomous")
	evaluator, _ := f.keyClient(t, "read", "urn:ietf:params:oauth:grant-type:token-exchange")
	f.request(t, "POST", "/api/admin/v1/mcp-resource-servers", map[string]any{"name": "Upstream", "resource": "https://upstream.example/mcp", "scopes": []string{"read"}}, f.admin, 201)
	rule := func(name string) map[string]any {
		return map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{"read"}, "token_exchange_resources": []string{"https://upstream.example/mcp"}, "tool": map[string]any{"name": name, "display_name": name}}
	}
	server := f.request(t, "POST", "/api/admin/v1/mcp-resource-servers", map[string]any{"name": "Managed documents", "scopes": []string{"read"}, "client_id": evaluator, "hosting": map[string]any{"upstream_url": "https://upstream.example/mcp", "authentication": "token_exchange"}, "access_rules": []any{rule("read_document"), rule("delete_document")}}, f.admin, 201)
	resource := setupString(t, server, "resource")
	if !strings.HasPrefix(resource, stack.Issuer+"/realms/default/mcp/") {
		t.Fatalf("non-canonical resource: %s", resource)
	}
	jwk := server["gateway_dpop_jwk"].(map[string]any)
	jkt, err := tokens_jose.ValidatePublicSigningJWK(jwk)
	if err != nil {
		t.Fatalf("private or invalid gateway key: %v", err)
	}
	tool := server["access_rules"].([]any)[0].(map[string]any)["tool"].(map[string]any)
	assignment := map[string]any{"resource_type": "idmagic_mcp_operation", "resource_id": tool["assignment_resource_id"], "relation": "invoke", "subject_type": "agent", "subject_id": agent}
	f.request(t, "POST", "/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": []any{assignment}}, f.admin, 200)
	issued := f.signedForm(t, key, agentClient, "/token", url.Values{"grant_type": {"client_credentials"}, "resource": {resource}}, 200)
	token := setupString(t, issued, "access_token")
	response := managedMcpCall(t, f, resource, token, "read_document", 200)
	if forwarded != 1 || downstreamToken == token || downstreamProof == "" {
		t.Fatalf("forwarded=%d, reused incoming credential=%v, proof missing=%v", forwarded, downstreamToken == token, downstreamProof == "")
	}
	claims := unverifiedClaims(t, downstreamToken)
	if claims["aud"] != "https://upstream.example/mcp" || claims["cnf"].(map[string]any)["jkt"] != jkt {
		t.Fatalf("wrong audience or key binding: %v", claims)
	}
	if response.Header().Get("Set-Cookie") != "" || response.Header().Get("Authorization") != "" {
		t.Fatal("upstream credentials were exposed")
	}
	// 本番と共通の経路分類と入場制御を通す。これは受付枠の局所検証であり、
	// 本番規模の混合負荷の SLO を測ったものではない。
	hold.Store(true)
	results := make(chan *httptest.ResponseRecorder, 8)
	for range 8 {
		go func() {
			body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"read_document","arguments":{},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
			req := httptest.NewRequest("POST", resource, strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Mcp-Protocol-Version", "2026-07-28")
			req.Header.Set("Mcp-Method", "tools/call")
			req.Header.Set("Mcp-Name", "read_document")
			rec := httptest.NewRecorder()
			f.s.Echo.ServeHTTP(rec, req)
			results <- rec
		}()
	}
	for range 8 {
		select {
		case <-opened:
		case <-time.After(3 * time.Second):
			finishStreams()
			t.Fatal("streams did not enter the gateway")
		}
	}
	// 長い接続が下位の枠を満たしていても、正式な token 発行の入口は受け付ける。
	f.signedForm(t, key, agentClient, "/token", url.Values{"grant_type": {"client_credentials"}, "resource": {resource}}, 200)
	managedMcpCall(t, f, resource, token, "read_document", 503)
	finishStreams()
	for range 8 {
		select {
		case rec := <-results:
			if rec.Code != 200 || !strings.Contains(rec.Body.String(), "data: waiting") {
				t.Fatalf("held stream: %d %s", rec.Code, rec.Body.String())
			}
		case <-time.After(3 * time.Second):
			t.Fatal("stream did not finish")
		}
	}
	hold.Store(false)
	for _, tc := range []struct {
		expression, arguments string
		want                  int
	}{{"false", "{}", 200}, {"true", "{}", 403}, {"attributes.missing > 0", "{}", 403}, {"attributes.amount > 100", `{"amount":5}`, 200}, {"attributes.amount > 100", `{"amount":150}`, 403}, {"false", `{"large":"` + strings.Repeat("x", 16<<10) + `"}`, 403}} {
		conditioned := rule("read_document")
		conditioned["condition"] = map[string]any{"deny_when": tc.expression}
		f.request(t, "PATCH", "/api/admin/v1/mcp-resource-servers/"+setupString(t, server, "id"), map[string]any{"access_rules": []any{conditioned, rule("delete_document")}}, f.admin, 200)
		expectedArguments = tc.arguments
		managedMcpCall(t, f, resource, token, "read_document", tc.want, tc.arguments)
	}
	if forwarded != 3 {
		t.Fatalf("condition evaluation forwarded %d requests, want 3", forwarded)
	}
	f.request(t, "PATCH", "/api/admin/v1/mcp-resource-servers/"+setupString(t, server, "id"), map[string]any{"access_rules": []any{rule("read_document"), rule("delete_document")}}, f.admin, 200)
	managedMcpCall(t, f, resource, token, "delete_document", 403)
	managedMcpCall(t, f, resource, token, "missing", 403)
	f.request(t, "POST", "/api/admin/v1/authorization/relation-tuples", map[string]any{"deletes": []any{assignment}}, f.admin, 200)
	managedMcpCall(t, f, resource, token, "read_document", 403)
	if forwarded != 3 {
		t.Fatalf("denied requests reached upstream: %d", forwarded)
	}
	for _, path := range []string{stack.Issuer + "/.well-known/oauth-protected-resource" + strings.TrimPrefix(resource, stack.Issuer), strings.Replace(resource, "/mcp/", "/.well-known/oauth-protected-resource/mcp/", 1)} {
		req := httptest.NewRequest("GET", path, http.NoBody)
		rec := httptest.NewRecorder()
		f.s.Echo.ServeHTTP(rec, req)
		var metadata map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &metadata); err != nil || rec.Code != 200 || metadata["resource"] != resource {
			t.Fatalf("metadata %s: %d %s error=%v", path, rec.Code, rec.Body.String(), err)
		}
	}
	managedMcpCall(t, f, strings.Replace(resource, "/realms/default/", "/realms/other/", 1), token, "read_document", 404)
	f.request(t, "PATCH", "/api/admin/v1/mcp-resource-servers/"+setupString(t, server, "id"), map[string]any{"state": "Disabled"}, f.admin, 200)
	managedMcpCall(t, f, resource, token, "read_document", 404)
	if forwarded != 3 {
		t.Fatal("disabled or cross-tenant request was forwarded")
	}
}
