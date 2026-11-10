package server_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ambi/idmagic/backend/mcpgateway"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

type approvalGatewayAudit struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (a *approvalGatewayAudit) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buffer.Write(p)
}

func (a *approvalGatewayAudit) String() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.buffer.String()
}

type mcpApprovalFixture struct {
	*approvedOperationFixture
	config         string
	audit          approvalGatewayAudit
	effects        atomic.Int32
	dropResponse   atomic.Bool
	dropEvaluation atomic.Bool
	inflight       sync.WaitGroup
}

type approvedToolInput struct {
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

func newMcpApprovalFixture(t *testing.T) *mcpApprovalFixture {
	t.Helper()
	f := &mcpApprovalFixture{approvedOperationFixture: newApprovedOperationFixture(t)}
	f.s.Echo.Use(support.RequestIDMiddleware(false))
	idmagic := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.dropEvaluation.Load() && strings.HasSuffix(r.URL.Path, "/resource-access/evaluate") {
			f.s.Echo.ServeHTTP(httptest.NewRecorder(), r)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		f.s.Echo.ServeHTTP(w, r)
	}))
	t.Cleanup(idmagic.Close)
	server := mcp.NewServer(&mcp.Implementation{Name: "ordinary-mcp-server", Version: "1"}, nil)
	for _, name := range []string{"execute", "other"} {
		mcp.AddTool(server, &mcp.Tool{Name: name}, func(context.Context, *mcp.CallToolRequest, approvedToolInput) (*mcp.CallToolResult, readDocumentOutput, error) {
			f.effects.Add(1)
			return nil, readDocumentOutput{Title: "executed"}, nil
		})
	}
	sdk := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if f.dropResponse.Load() {
			// SDK がツールを実行してから、クライアントへ応答を一バイトも返さずに接続を閉じる。
			sdk.ServeHTTP(httptest.NewRecorder(), r)
			connection, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = connection.Close()
			return
		}
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
	f.config = write("gateway.yaml", fmt.Appendf(nil, `listen: 127.0.0.1:0
resource: %s
upstream: %s/
idmagic:
  realm_url: %s/realms/default
  ca_file: %s
  client_id: %s
  client_secret_file: %s
mapping:
  tools:
    execute: {resource_type: document, relation: viewer, resource_id: d1, attribute_arguments: [amount]}
    other: {resource_type: document, relation: viewer, resource_id: d2}
`, agentRevocationResource, upstream.URL, idmagic.URL,
		write("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})),
		evalResourceServerClient, write("secret", []byte(evalResourceServerSecret))))
	return f
}

func (f *mcpApprovalFixture) gateway(t *testing.T) *httptest.Server {
	t.Helper()
	loaded, err := mcpgateway.Load(f.config, &f.audit)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.inflight.Add(1)
		defer f.inflight.Done()
		loaded.Handler().ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func (f *mcpApprovalFixture) call(t *testing.T, gateway *httptest.Server, token, tool, parameters string, id int) (int, string) {
	t.Helper()
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":%q,"arguments":%s,"_meta":{"io.modelcontextprotocol/protocolVersion":%q,"io.modelcontextprotocol/clientCapabilities":{}}}}`, id, tool, parameters, mcpProtocolVersion)
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, gateway.URL+"/", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Mcp-Protocol-Version", mcpProtocolVersion)
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", tool)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(raw)
}

func requireApprovalGatewayError(t *testing.T, status int, body string, wantStatus, wantCode int) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatalf("error body=%s: %v", body, err)
	}
	if status != wantStatus || envelope.Error.Code != wantCode {
		t.Fatalf("status=%d body=%s; want status=%d code=%d", status, body, wantStatus, wantCode)
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-MCPGATEWAY-006, REQ-OAUTH2-065: 対象と引数が一致する承認を一回だけ転送し、再送は拒否し、承認の相関だけを記録すること
func TestMcpGatewayForwardsAnApprovalAtMostOnce(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	token := f.approvedToken(t)
	status, body := f.call(t, gateway, token, "execute", `{"amount":101,"currency":"JPY"}`, 1)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32000)
	status, body = f.call(t, gateway, token, "other", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32000)
	if f.effects.Load() != 0 {
		t.Fatalf("mismatches executed: %d", f.effects.Load())
	}
	status, body = f.call(t, gateway, token, "execute", `{"currency":"JPY", "amount":100}`, 3)
	if status != http.StatusOK || !strings.Contains(body, "executed") || f.effects.Load() != 1 {
		t.Fatalf("approved call: status=%d body=%s effects=%d", status, body, f.effects.Load())
	}
	for _, id := range []int{3, 4} {
		status, body = f.call(t, gateway, token, "execute", approvedParameters, id)
		requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	}
	// 同じ引数でも、新たな人の承認があれば別の一回を実行できる。
	secondToken := f.approvedToken(t)
	status, body = f.call(t, gateway, secondToken, "execute", approvedParameters, 5)
	if status != http.StatusOK || f.effects.Load() != 2 {
		t.Fatalf("new approval: %d %s effects=%d", status, body, f.effects.Load())
	}
	f.inflight.Wait()
	log := f.audit.String()
	if !strings.Contains(log, `"approval_request_id":`) || !strings.Contains(log, `"operation_digest":`) {
		t.Fatalf("missing approval correlation: %s", log)
	}
	if strings.Contains(log, token) || strings.Contains(log, `"amount"`) || strings.Contains(log, `"operation_key"`) {
		t.Fatalf("credentials or arguments logged: %s", log)
	}
}

//spec:covers REQ-MCPGATEWAY-013: 別プロセス相当のゲートウェイへの並行要求と再作成で承認を再利用しないこと
func TestMcpGatewaySharesApprovalUseAcrossInstances(t *testing.T) {
	f := newMcpApprovalFixture(t)
	first, second := f.gateway(t), f.gateway(t)
	token := f.approvedToken(t)
	start := make(chan struct{})
	statuses := make(chan int, 2)
	for i, gateway := range []*httptest.Server{first, second} {
		go func() {
			status := 0
			defer func() { statuses <- status }()
			<-start
			status, _ = f.call(t, gateway, token, "execute", approvedParameters, i+1)
		}()
	}
	close(start)
	counts := map[int]int{}
	counts[<-statuses]++
	counts[<-statuses]++
	if counts[http.StatusOK] != 1 || counts[http.StatusForbidden] != 1 || f.effects.Load() != 1 {
		t.Fatalf("parallel calls: statuses=%v effects=%d", counts, f.effects.Load())
	}
	first.Close()
	second.Close()
	restarted := f.gateway(t)
	status, body := f.call(t, restarted, token, "execute", approvedParameters, 3)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 1 {
		t.Fatalf("restart executed: %d", f.effects.Load())
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-MCPGATEWAY-006: 実行後に上流の応答が失われても使用を戻さず、結果不明を返して再送を拒否すること
func TestMcpGatewayDoesNotRetryAnUnknownApprovedOutcome(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	token := f.approvedToken(t)
	f.dropResponse.Store(true)
	status, body := f.call(t, gateway, token, "execute", approvedParameters, 1)
	requireApprovalGatewayError(t, status, body, http.StatusBadGateway, -32005)
	if !strings.Contains(body, `"id":1`) {
		t.Fatalf("lost RPC correlation: %s", body)
	}
	if f.effects.Load() != 1 {
		t.Fatalf("lost response effects=%d", f.effects.Load())
	}
	f.dropResponse.Store(false)
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 1 {
		t.Fatalf("retry executed: %d", f.effects.Load())
	}
	f.inflight.Wait()
	if !strings.Contains(f.audit.String(), `"outcome":"approval_outcome_unknown"`) {
		t.Fatalf("audit=%s", f.audit.String())
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-OAUTH2-065: 使用を記録した評価の応答を失っても許可を推測せず、承認を再発行しないこと
func TestMcpGatewayDoesNotReleaseApprovalWhenEvaluationResponseIsLost(t *testing.T) {
	f := newMcpApprovalFixture(t)
	gateway := f.gateway(t)
	token := f.approvedToken(t)
	f.dropEvaluation.Store(true)
	status, body := f.call(t, gateway, token, "execute", approvedParameters, 1)
	requireApprovalGatewayError(t, status, body, http.StatusServiceUnavailable, -32002)
	f.dropEvaluation.Store(false)
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 0 {
		t.Fatalf("unconfirmed permit was forwarded: %d", f.effects.Load())
	}
}

//spec:covers REQ-MCPGATEWAY-013, REQ-OAUTH2-065: 条件が承認を求める場合だけ承認を消費し、閾値以下は通常の認可で実行できること
func TestMcpGatewayClaimsApprovalOnlyWhenTheConditionRequiresIt(t *testing.T) {
	f := newMcpApprovalFixture(t)
	if status, body := f.putCondition(t, map[string]any{
		"require_approval_when": "attributes.amount >= 100", "approval_detail_type": approvedDetailType,
	}); status != http.StatusOK {
		t.Fatalf("condition: status=%d body=%v", status, body)
	}
	gateway := f.gateway(t)
	ordinaryToken := f.resourceToken(t, f.clientID, agentRevocationResource, nil)
	status, body := f.call(t, gateway, ordinaryToken, "execute", `{"amount":80,"currency":"JPY"}`, 1)
	if status != http.StatusOK {
		t.Fatalf("ordinary call: %d %s", status, body)
	}
	status, body = f.call(t, gateway, ordinaryToken, "execute", approvedParameters, 2)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32000)
	token := f.approvedToken(t)
	status, body = f.call(t, gateway, token, "execute", `{"amount":80,"currency":"JPY"}`, 3)
	if status != http.StatusOK {
		t.Fatalf("approval not needed: %d %s", status, body)
	}
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 4)
	if status != http.StatusOK {
		t.Fatalf("approval was consumed below threshold: %d %s", status, body)
	}
	status, body = f.call(t, gateway, token, "execute", approvedParameters, 5)
	requireApprovalGatewayError(t, status, body, http.StatusForbidden, -32004)
	if f.effects.Load() != 3 {
		t.Fatalf("effects=%d", f.effects.Load())
	}
}
