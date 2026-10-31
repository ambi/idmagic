package server_http_test

// MCP ゲートウェイを、製品の評価の経路と実在の MCP SDK のサーバーの間に置いて確かめる。
// 上流のサーバーはゲートウェイを知らず、保証はツールの実行回数で読む。

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ambi/idmagic/backend/mcpgateway"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const mcpProtocolVersion = "2026-07-28"

type mcpGatewayFixture struct {
	*resourceAccessFixture
	idmagic  *httptest.Server
	gateway  *httptest.Server
	audit    *bytes.Buffer
	effects  atomic.Int32
	deleted  atomic.Int32
	upstream atomic.Value
}

type readDocumentInput struct {
	DocumentID string `json:"document_id"`
}

type readDocumentOutput struct {
	Title string `json:"title"`
}

func newMcpGatewayFixture(t *testing.T) *mcpGatewayFixture {
	t.Helper()
	f := &mcpGatewayFixture{resourceAccessFixture: newResourceAccessFixture(t, "A1"), audit: &bytes.Buffer{}}
	f.grantD1(t)
	// 本番の cmd/idmagic と同じく要求 ID を入口で付ける。受信した値は信頼しない。
	f.s.Echo.Use(support.RequestIDMiddleware(false))
	f.idmagic = httptest.NewTLSServer(f.s.Echo)
	t.Cleanup(f.idmagic.Close)

	server := mcp.NewServer(&mcp.Implementation{Name: "unmodified", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "read_document"}, func(context.Context, *mcp.CallToolRequest, readDocumentInput) (*mcp.CallToolResult, readDocumentOutput, error) {
		f.effects.Add(1)
		return nil, readDocumentOutput{Title: "d1"}, nil
	})
	mcp.AddTool(server, &mcp.Tool{Name: "delete_document"}, func(context.Context, *mcp.CallToolRequest, readDocumentInput) (*mcp.CallToolResult, readDocumentOutput, error) {
		f.deleted.Add(1)
		return nil, readDocumentOutput{}, nil
	})
	sdk := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.upstream.Store(r.Header.Clone())
		sdk.ServeHTTP(w, r)
	}))
	t.Cleanup(upstream.Close)

	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	config := write("config.yaml", fmt.Appendf(nil, `listen: 127.0.0.1:0
resource: %s
upstream: %s/
idmagic:
  realm_url: %s/realms/default
  ca_file: %s
  client_id: %s
  client_secret_file: %s
mapping:
  methods:
    server/discover: { resource_type: document, relation: viewer, resource_id: d1 }
    tools/list: { resource_type: document, relation: viewer, resource_id: d1 }
  tools:
    read_document: { resource_type: document, relation: viewer, resource_id_argument: document_id }
`, agentRevocationResource, upstream.URL, f.idmagic.URL,
		write("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.idmagic.Certificate().Raw})),
		evalResourceServerClient, write("secret", []byte(evalResourceServerSecret+"\n"))))
	gateway, err := mcpgateway.Load(config, f.audit)
	if err != nil {
		t.Fatal(err)
	}
	f.gateway = httptest.NewServer(gateway.Handler())
	t.Cleanup(f.gateway.Close)
	return f
}

type presentToken struct{ token string }

func (p presentToken) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Authorization", "Bearer "+p.token)
	return http.DefaultTransport.RoundTrip(request)
}

func (f *mcpGatewayFixture) connect(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "agent", Version: "1"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint: f.gateway.URL + "/", HTTPClient: &http.Client{Transport: presentToken{token}},
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// post は一つの tools/call を SDK を通さずに送る。
func (f *mcpGatewayFixture) post(t *testing.T, tool string, change func(*http.Request)) (*http.Response, string) {
	t.Helper()
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":%q,"arguments":{"document_id":"d1"},`+
		`"_meta":{"io.modelcontextprotocol/protocolVersion":%q,"io.modelcontextprotocol/clientCapabilities":{}}}}`, tool, mcpProtocolVersion)
	return f.send(t, http.MethodPost, body, func(request *http.Request) {
		request.Header.Set("Mcp-Protocol-Version", mcpProtocolVersion)
		request.Header.Set("Mcp-Method", "tools/call")
		request.Header.Set("Mcp-Name", tool)
		if change != nil {
			change(request)
		}
	})
}

func (f *mcpGatewayFixture) send(t *testing.T, method, body string, change func(*http.Request)) (*http.Response, string) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), method, f.gateway.URL+"/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	if change != nil {
		change(request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	var text strings.Builder
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		text.WriteString(scanner.Text())
	}
	return response, text.String()
}

func gatewayBearer(token string) func(*http.Request) {
	return func(request *http.Request) { request.Header.Set("Authorization", "Bearer "+token) }
}

// auditLines はゲートウェイの監査ログを読む。
func (f *mcpGatewayFixture) auditLines(t *testing.T) []map[string]any {
	t.Helper()
	var lines []map[string]any
	scanner := bufio.NewScanner(bytes.NewReader(f.audit.Bytes()))
	for scanner.Scan() {
		line := map[string]any{}
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			t.Fatalf("audit line %q: %v", scanner.Text(), err)
		}
		lines = append(lines, line)
	}
	return lines
}

func (f *mcpGatewayFixture) lastEvaluated(t *testing.T) *oauthdomain.ResourceAccessEvaluated {
	t.Helper()
	for _, recorded := range slices.Backward(f.s.Events.All()) {
		if event, ok := recorded.(*oauthdomain.ResourceAccessEvaluated); ok {
			return event
		}
	}
	t.Fatal("ResourceAccessEvaluated が発行されていない")
	return nil
}

// 無改修のサーバーのツールは、許可した呼び出しでだけ実行される。評価の通信が切れたら実行されない。
//
//spec:covers REQ-MCPGATEWAY-003, REQ-MCPGATEWAY-004, EX-MCPGATEWAY-003-01, EX-MCPGATEWAY-003-02: SDK の実クライアントと実サーバーの間で、許可、関係の不許可、宣言のないツール、評価不能のツールの実行回数と、上流へトークンを渡さないこと
func TestMcpGatewayRunsOnlyPermittedToolCalls(t *testing.T) {
	f := newMcpGatewayFixture(t)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)
	session := f.connect(t, token)

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "read_document", Arguments: readDocumentInput{DocumentID: "d1"}})
	if err != nil || result.IsError || f.effects.Load() != 1 {
		t.Fatalf("許可の対照: result=%+v err=%v effects=%d", result, err, f.effects.Load())
	}
	headers := f.upstream.Load().(http.Header)
	if headers.Get("Authorization") != "" {
		t.Fatalf("上流へトークンが渡った: %v", headers)
	}

	if _, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "read_document", Arguments: readDocumentInput{DocumentID: "d2"}}); err == nil {
		t.Fatal("権限のない資源が成功した")
	}
	if got := f.effects.Load(); got != 1 {
		t.Fatalf("権限のない資源で実行された: %d", got)
	}

	if _, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "delete_document", Arguments: readDocumentInput{DocumentID: "d1"}}); err == nil {
		t.Fatal("宣言のないツールが成功した")
	}
	if got := f.deleted.Load(); got != 0 {
		t.Fatalf("宣言のないツールが実行された: %d", got)
	}

	f.idmagic.Close()
	response, body := f.post(t, "read_document", gatewayBearer(token))
	if response.StatusCode != http.StatusServiceUnavailable || f.effects.Load() != 1 {
		t.Fatalf("評価不能: status=%d body=%s effects=%d", response.StatusCode, body, f.effects.Load())
	}
}

// ゲートウェイのログと IdMagic の記録は同じ ID で結ばれ、主体は検証できた呼び出しにだけ残る。
//
//spec:covers REQ-OAUTH2-062, REQ-MCPGATEWAY-006: 宣言のないツールの試行者、評価の X-Request-ID と requestId の一致、ログに資格情報がないこと、証明のないトークンと別の audience の主体を残さないこと
func TestMcpGatewayAuditJoinsToVerifiedPrincipal(t *testing.T) {
	f := newMcpGatewayFixture(t)
	token := f.resourceToken(t, agentClientID("A1"), agentRevocationResource, nil)

	response, _ := f.post(t, "delete_document", gatewayBearer(token))
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d", response.StatusCode)
	}
	event := f.lastEvaluated(t)
	if event.AgentID != "A1" || event.Permitted || len(event.Reasons) != 1 || event.Reasons[0] != oauthdomain.ReasonOperationNotRegistered {
		t.Fatalf("宣言のないツールの試行者が残らない: %+v", event)
	}
	lines := f.auditLines(t)
	line := lines[len(lines)-1]
	if line["kind"] != "mcp_gateway_request" || line["tool"] != "delete_document" || line["outcome"] != "denied" ||
		line["correlation_id"] != response.Header.Get("X-Request-ID") || line["correlation_id"] == "" {
		t.Fatalf("audit line: %v (X-Request-ID %q)", line, response.Header.Get("X-Request-ID"))
	}
	if event.RequestID == "" || line["evaluation_request_id"] != event.RequestID {
		t.Fatalf("相関できない: log=%v event=%q", line["evaluation_request_id"], event.RequestID)
	}
	if strings.Contains(f.audit.String(), token) {
		t.Fatal("ログにトークンがある")
	}

	now := time.Now().UTC()
	if err := f.s.McpResourceServers.Save(context.Background(), &oauthdomain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ID: "mcp-other", Resource: evalOtherResource,
		Name: "other", Scopes: []string{"tools:read"}, State: oauthdomain.McpResourceServerActive,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	other := f.resourceToken(t, agentClientID("A1"), evalOtherResource, nil)
	if response, _ := f.post(t, "read_document", gatewayBearer(other)); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("別の audience: status=%d", response.StatusCode)
	}
	if event := f.lastEvaluated(t); event.AgentID != "" || event.UserID != "" {
		t.Fatalf("拒否したトークンの主体が残った: %+v", event)
	}

	key, jwk, _ := tpDPoPKey(t)
	bound := f.dpopToken(t, key, jwk)
	if response, _ := f.post(t, "read_document", gatewayBearer(bound)); response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("証明なし: status=%d", response.StatusCode)
	}
	if event := f.lastEvaluated(t); event.AgentID != "" || event.UserID != "" || event.Reasons[0] != oauthdomain.ReasonSenderProofInvalid {
		t.Fatalf("証明のないトークンの持ち主が残った: %+v", event)
	}
}

func (f *mcpGatewayFixture) dpopToken(t *testing.T, key *rsa.PrivateKey, jwk map[string]any) string {
	t.Helper()
	return f.resourceToken(t, agentClientID("A1"), agentRevocationResource, func(request *http.Request) {
		request.Header.Set("DPoP", tpDPoPProof(t, key, jwk, tpDPoPProofInput{
			htm: http.MethodPost, htu: stack.Issuer + "/realms/default/token", jti: "issue", iat: time.Now(),
		}))
	})
}

// 再送は毎回評価し、証明の再利用、別の audience、解釈できない要求は上流へ届かない。
//
//spec:covers REQ-MCPGATEWAY-002, REQ-MCPGATEWAY-003, REQ-MCPGATEWAY-005, REQ-OAUTH2-063: DPoP の公開 URL を htu にした評価、証明の再利用、バッチ、古い方法、未認証の 401 と保護リソースのメタデータ
func TestMcpGatewayRejectsWhatItCannotAuthorize(t *testing.T) {
	f := newMcpGatewayFixture(t)
	key, jwk, _ := tpDPoPKey(t)
	token := f.dpopToken(t, key, jwk)
	digest := sha256.Sum256([]byte(token))
	proof := func(jti string) string {
		return tpDPoPProof(t, key, jwk, tpDPoPProofInput{
			htm: http.MethodPost, htu: agentRevocationResource, jti: jti, iat: time.Now(),
			ath: base64.RawURLEncoding.EncodeToString(digest[:]),
		})
	}
	withProof := func(value string) func(*http.Request) {
		return func(request *http.Request) {
			request.Header.Set("Authorization", "DPoP "+token)
			request.Header.Set("DPoP", value)
		}
	}
	first := proof("first")
	if response, body := f.post(t, "read_document", withProof(first)); response.StatusCode != http.StatusOK || f.effects.Load() != 1 {
		t.Fatalf("許可の対照: status=%d body=%s effects=%d", response.StatusCode, body, f.effects.Load())
	}
	if response, _ := f.post(t, "read_document", withProof(first)); response.StatusCode != http.StatusUnauthorized || f.effects.Load() != 1 {
		t.Fatalf("証明の再利用: status=%d effects=%d", response.StatusCode, f.effects.Load())
	}
	if response, _ := f.post(t, "read_document", withProof(proof("second"))); response.StatusCode != http.StatusOK || f.effects.Load() != 2 {
		t.Fatalf("新しい証明の再送: status=%d effects=%d", response.StatusCode, f.effects.Load())
	}

	response, _ := f.post(t, "read_document", nil)
	challenge := response.Header.Get("WWW-Authenticate")
	if response.StatusCode != http.StatusUnauthorized || !strings.Contains(challenge, `resource_metadata="https://mcp.agent-revocation.example/.well-known/oauth-protected-resource"`) {
		t.Fatalf("未認証: status=%d challenge=%q", response.StatusCode, challenge)
	}
	metadata, body := f.send(t, http.MethodGet, "", nil)
	if metadata.StatusCode != http.StatusMethodNotAllowed || metadata.Header.Get("Allow") != http.MethodPost {
		t.Fatalf("GET: status=%d body=%s", metadata.StatusCode, body)
	}

	batch := `[{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read_document","arguments":{"document_id":"d1"},"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}]`
	before := len(f.s.Events.All())
	if response, _ := f.send(t, http.MethodPost, batch, func(request *http.Request) {
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("Mcp-Protocol-Version", mcpProtocolVersion)
		request.Header.Set("Mcp-Method", "tools/call")
		request.Header.Set("Mcp-Name", "read_document")
	}); response.StatusCode != http.StatusBadRequest || f.effects.Load() != 2 {
		t.Fatalf("バッチ: status=%d effects=%d", response.StatusCode, f.effects.Load())
	}
	if got := len(f.s.Events.All()); got != before {
		t.Fatalf("バッチで評価された: %d -> %d", before, got)
	}
}
