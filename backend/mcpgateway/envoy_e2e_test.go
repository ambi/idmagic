package mcpgateway_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/mcpgateway"
)

const (
	envoyResource   = "https://mcp.example/mcp"
	envoyBodyLimit  = 4194304
	envoyGoodToken  = "good-token"
	envoyLimitToken = "limited-token"
)

// envoyFixture は、配布する設定例を読み込んだ実在の Envoy と、その背後のゲートウェイ、評価のダブル、上流である。
type envoyFixture struct {
	url             string
	gateway         *httptest.Server
	evaluations     atomic.Int64
	proofs          atomic.Int64
	upstreamMu      sync.Mutex
	upstream        []upstreamCall
	upstreamHandler atomic.Pointer[http.HandlerFunc]
}

type upstreamCall struct {
	uri     string
	headers http.Header
	body    string
}

func listenOnAllInterfaces(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(handler)
	if err := server.Listener.Close(); err != nil {
		t.Fatal(err)
	}
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)
	return server
}

func serverPort(t *testing.T, server *httptest.Server) int {
	t.Helper()
	return server.Listener.Addr().(*net.TCPAddr).Port
}

func docker(t *testing.T, args ...string) string {
	t.Helper()
	// 標準出力だけを結果にする。docker create はイメージの取得の経過を標準エラーへ書く。
	var stderr bytes.Buffer
	command := exec.Command("docker", args...)
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(string(output))
}

// configForFixture は設定例の接続先の待ち受けだけを試験のものへ替える。ほかの値を変えると設定例の試験でなくなる。
func configForFixture(t *testing.T, gatewayPort, upstreamPort int) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "envoy-mcp-gateway", "envoy.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, port := range []struct {
		from string
		to   int
	}{{"port_value: 8090", gatewayPort}, {"port_value: 8091", upstreamPort}} {
		if !bytes.Contains(raw, []byte(port.from)) {
			t.Fatalf("the example has no %q", port.from)
		}
		raw = bytes.ReplaceAll(raw, []byte(port.from), fmt.Appendf(nil, "port_value: %d", port.to))
	}
	return raw
}

func newEnvoyFixture(t *testing.T) *envoyFixture {
	t.Helper()
	image := os.Getenv("ENVOY_IMAGE")
	if image == "" {
		t.Skip("run mise run test-envoy-mcp-gateway to start a real Envoy")
	}
	// Envoy の状態は Go の結果キャッシュの鍵に入らないため、実行ごとの値で再利用を防ぐ。
	_ = os.Getenv("VERIFICATION_RUN_ID")
	f := &envoyFixture{}

	idmagic := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.evaluations.Add(1)
		if r.FormValue("dpop_proof") != "" {
			f.proofs.Add(1)
		}
		w.Header().Set("X-Request-ID", "evaluation-id")
		if r.FormValue("token") == envoyLimitToken {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		permitted := r.FormValue("token") == envoyGoodToken && (r.FormValue("resource_id") == "d1" || r.FormValue("resource_type") == "mcp_tool")
		_ = json.NewEncoder(w).Encode(map[string]any{"decision": permitted, "reasons": []string{}})
	}))
	t.Cleanup(idmagic.Close)

	upstream := listenOnAllInterfaces(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.upstreamMu.Lock()
		f.upstream = append(f.upstream, upstreamCall{uri: r.URL.RequestURI(), headers: r.Header.Clone(), body: string(body)})
		f.upstreamMu.Unlock()
		w.Header().Set("X-Request-ID", "upstream-id")
		if handler := f.upstreamHandler.Load(); handler != nil {
			(*handler)(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{}}`)
	}))

	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	config := write("gateway.yaml", fmt.Appendf(nil, `mode: envoy_ext_authz
listen: 127.0.0.1:0
resource: %s
idmagic:
  realm_url: %s/realms/default
  ca_file: %s
  client_id: gateway
  client_secret_file: %s
allowed_origins: [https://allowed.example]
mapping:
  methods:
    tools/list: { resource_type: mcp_server, relation: list_tools, resource_id: d1 }
  tools:
    read_document: { resource_type: document, relation: viewer, resource_id_argument: document_id }
`, envoyResource, idmagic.URL,
		write("ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: idmagic.Certificate().Raw})),
		write("secret", []byte("secret\n"))))
	gateway, err := mcpgateway.Load(config, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	f.gateway = listenOnAllInterfaces(t, gateway.Handler())

	envoyConfig := write("envoy.yaml", configForFixture(t, serverPort(t, f.gateway), serverPort(t, upstream)))
	container := docker(t, "create", "--add-host", "mcp-gateway:host-gateway", "--add-host", "mcp-server:host-gateway",
		"--publish", "127.0.0.1::10000", image, "-c", "/etc/envoy/envoy.yaml", "-l", "warn")
	t.Cleanup(func() {
		if t.Failed() {
			output, _ := exec.Command("docker", "logs", container).CombinedOutput()
			t.Logf("envoy logs:\n%s", output)
		}
		_ = exec.Command("docker", "rm", "-f", container).Run()
	})
	docker(t, "cp", envoyConfig, container+":/etc/envoy/envoy.yaml")
	docker(t, "start", container)
	f.url = "http://" + docker(t, "port", container, "10000/tcp")

	// メタデータの経路は評価と上流を呼ばないので、起動の待ちに使っても記録を汚さない。
	deadline := time.Now().Add(30 * time.Second)
	for {
		response, err := http.Get(f.url + "/.well-known/oauth-protected-resource")
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return f
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("envoy did not become ready: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (f *envoyFixture) upstreamCalls() []upstreamCall {
	f.upstreamMu.Lock()
	defer f.upstreamMu.Unlock()
	return append([]upstreamCall(nil), f.upstream...)
}

func toolCallBody(tool, documentID string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":%q,"arguments":{"document_id":%q},`+
		`"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`, tool, documentID)
}

func toolCall(t *testing.T, target, tool, documentID string, change func(*http.Request)) *http.Response {
	t.Helper()
	return postBody(t, target, tool, toolCallBody(tool, documentID), change)
}

func postBody(t *testing.T, target, tool, body string, change func(*http.Request)) *http.Response {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+envoyGoodToken)
	request.Header.Set("Mcp-Protocol-Version", "2026-07-28")
	request.Header.Set("Mcp-Method", "tools/call")
	request.Header.Set("Mcp-Name", tool)
	if change != nil {
		change(request)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func rpcCode(t *testing.T, response *http.Response) int {
	t.Helper()
	var envelope struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("response is not a JSON-RPC error: %v", err)
	}
	return envelope.Error.Code
}

//spec:covers REQ-MCPGATEWAY-007, EX-MCPGATEWAY-007-01: 実在の Envoy の背後で、許可だけが資格情報とクエリなしで上流へ届き、拒否はゲートウェイの応答のまま返る
func TestEnvoyForwardsOnlyPermittedCalls(t *testing.T) {
	f := newEnvoyFixture(t)
	target := f.url + "/mcp"

	permitted := toolCall(t, target, "read_document", "d1", func(r *http.Request) {
		r.Header.Set("DPoP", "proof")
		r.Header.Set("Cookie", "session=1")
		r.Header.Set("X-Request-ID", "client-chosen")
	})
	if permitted.StatusCode != http.StatusOK {
		t.Fatalf("permitted status = %d", permitted.StatusCode)
	}
	calls := f.upstreamCalls()
	if len(calls) != 1 {
		t.Fatalf("upstream calls = %d, want 1", len(calls))
	}
	call := calls[0]
	if call.body != toolCallBody("read_document", "d1") || call.uri != "/mcp" {
		t.Fatalf("forwarded %s %q", call.uri, call.body)
	}
	for _, name := range []string{"Authorization", "Dpop", "Cookie"} {
		if call.headers.Get(name) != "" {
			t.Fatalf("%s reached the upstream", name)
		}
	}
	if f.proofs.Load() != 1 {
		t.Fatalf("the DPoP proof did not reach the evaluation")
	}
	correlation := permitted.Header.Values("X-Request-ID")
	if len(correlation) != 1 || len(correlation[0]) != 32 || correlation[0] != call.headers.Get("X-Request-ID") {
		t.Fatalf("client saw X-Request-ID %q, upstream saw %q", correlation, call.headers.Values("X-Request-ID"))
	}

	cases := []struct {
		name          string
		send          func() *http.Response
		status        int
		code          int
		header        string
		headerValue   string
		noEvaluations bool
	}{
		{
			name:   "denied by the evaluation",
			send:   func() *http.Response { return toolCall(t, target, "read_document", "d2", nil) },
			status: http.StatusForbidden, code: -32000,
		},
		{
			name: "no token",
			send: func() *http.Response {
				return toolCall(t, target, "read_document", "d1", func(r *http.Request) { r.Header.Del("Authorization") })
			},
			status: http.StatusUnauthorized, code: -32001, header: "WWW-Authenticate", noEvaluations: true,
			headerValue: `Bearer resource_metadata="https://mcp.example/.well-known/oauth-protected-resource/mcp"`,
		},
		{
			name: "two authorization headers joined by envoy",
			send: func() *http.Response {
				return toolCall(t, target, "read_document", "d1", func(r *http.Request) { r.Header.Add("Authorization", "Bearer other") })
			},
			status: http.StatusUnauthorized, code: -32001, noEvaluations: true,
		},
		{
			name: "two proofs joined by envoy",
			send: func() *http.Response {
				return toolCall(t, target, "read_document", "d1", func(r *http.Request) {
					r.Header.Add("DPoP", "one")
					r.Header.Add("DPoP", "two")
				})
			},
			status: http.StatusUnauthorized, code: -32001, noEvaluations: true,
		},
		{
			name: "mirrored header differs from the body",
			send: func() *http.Response {
				return toolCall(t, target, "read_document", "d1", func(r *http.Request) { r.Header.Set("Mcp-Name", "other") })
			},
			status: http.StatusBadRequest, code: -32020, noEvaluations: true,
		},
		{
			name:   "tool without a declaration is evaluated and not forwarded",
			send:   func() *http.Response { return toolCall(t, target, "delete_document", "d1", nil) },
			status: http.StatusForbidden, code: -32000,
		},
		{
			name: "rate limited evaluation stays closed without Retry-After",
			send: func() *http.Response {
				return toolCall(t, target, "read_document", "d1", func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+envoyLimitToken) })
			},
			status: http.StatusServiceUnavailable, code: -32002,
		},
		{
			name: "origin outside the allowed list reaches the gateway check",
			send: func() *http.Response {
				return toolCall(t, target, "read_document", "d1", func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") })
			},
			status: http.StatusForbidden, noEvaluations: true,
		},
		{
			name:   "query",
			send:   func() *http.Response { return toolCall(t, target+"?token=secret", "read_document", "d1", nil) },
			status: http.StatusNotFound, noEvaluations: true,
		},
		{
			name: "method other than POST",
			send: func() *http.Response {
				response, err := http.Get(target)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = response.Body.Close() })
				return response
			},
			status: http.StatusMethodNotAllowed, header: "Allow", headerValue: http.MethodPost, noEvaluations: true,
		},
		{
			name:   "path without a route",
			send:   func() *http.Response { return toolCall(t, f.url+"/other", "read_document", "d1", nil) },
			status: http.StatusNotFound, noEvaluations: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			evaluations := f.evaluations.Load()
			response := c.send()
			if response.StatusCode != c.status {
				t.Fatalf("status = %d, want %d", response.StatusCode, c.status)
			}
			if c.code != 0 {
				if code := rpcCode(t, response); code != c.code {
					t.Fatalf("code = %d, want %d", code, c.code)
				}
			}
			if c.header != "" && response.Header.Get(c.header) != c.headerValue {
				t.Fatalf("%s = %q", c.header, response.Header.Get(c.header))
			}
			if c.noEvaluations && f.evaluations.Load() != evaluations {
				t.Fatal("the evaluation was called")
			}
			if len(f.upstreamCalls()) != 1 {
				t.Fatalf("upstream calls = %d, want the single permitted call", len(f.upstreamCalls()))
			}
		})
	}

	t.Run("metadata comes from the gateway", func(t *testing.T) {
		evaluations := f.evaluations.Load()
		response, err := http.Get(f.url + "/.well-known/oauth-protected-resource/mcp")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		var metadata struct {
			Resource string `json:"resource"`
		}
		if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusOK || metadata.Resource != envoyResource {
			t.Fatalf("metadata = %d %+v", response.StatusCode, metadata)
		}
		if f.evaluations.Load() != evaluations || len(f.upstreamCalls()) != 1 {
			t.Fatal("metadata reached the evaluation or the upstream")
		}
	})
}

//spec:covers REQ-MCPGATEWAY-007: ゲートウェイの停止と本文の上限の超過で上流を呼ばない。上限ちょうどの本文と chunked の本文は同じバイト列で届く
func TestEnvoyFailsClosed(t *testing.T) {
	f := newEnvoyFixture(t)
	target := f.url + "/mcp"
	sized := func(size int) string {
		base := toolCallBody("read_document", "d1")
		padding := size - len(base) - len(`"p":"",`)
		return strings.Replace(base, `"jsonrpc"`, `"p":"`+strings.Repeat("x", padding)+`","jsonrpc"`, 1)
	}

	t.Run("body at the limit is forwarded", func(t *testing.T) {
		body := sized(envoyBodyLimit)
		if len(body) != envoyBodyLimit {
			t.Fatalf("body has %d bytes", len(body))
		}
		response := postBody(t, target, "read_document", body, nil)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", response.StatusCode)
		}
		calls := f.upstreamCalls()
		if len(calls) != 1 || calls[0].body != body {
			t.Fatalf("upstream calls = %d", len(calls))
		}
	})

	t.Run("chunked body is forwarded whole", func(t *testing.T) {
		body := toolCallBody("read_document", "d1")
		response := postBody(t, target, "read_document", body, func(r *http.Request) {
			r.Body = io.NopCloser(strings.NewReader(body))
			r.GetBody = nil
			r.ContentLength = -1
		})
		if response.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", response.StatusCode)
		}
		calls := f.upstreamCalls()
		if len(calls) != 2 || calls[1].body != body {
			t.Fatalf("upstream calls = %d", len(calls))
		}
	})

	t.Run("body over the limit reaches nothing", func(t *testing.T) {
		evaluations := f.evaluations.Load()
		body := sized(envoyBodyLimit) + " "
		response := postBody(t, target, "read_document", body, nil)
		if response.StatusCode != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d", response.StatusCode)
		}
		if f.evaluations.Load() != evaluations || len(f.upstreamCalls()) != 2 {
			t.Fatal("an oversized body reached the evaluation or the upstream")
		}
	})

	t.Run("gateway outage reaches nothing", func(t *testing.T) {
		f.gateway.Close()
		response := toolCall(t, target, "read_document", "d1", nil)
		if response.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("status = %d", response.StatusCode)
		}
		if code := rpcCode(t, response); code != -32002 {
			t.Fatalf("code = %d", code)
		}
		if len(f.upstreamCalls()) != 2 {
			t.Fatal("the request reached the upstream without a decision")
		}
	})
}

//spec:covers REQ-MCPGATEWAY-007: SSE の各イベントを上流が次を書く前に受け取り、MCP クライアントの切断で上流の要求が取り消される
func TestEnvoyStreamsEventsAndPropagatesDisconnect(t *testing.T) {
	f := newEnvoyFixture(t)
	firstRead := make(chan struct{})
	canceled := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-firstRead:
		case <-time.After(10 * time.Second):
			t.Error("the first event was held back until the stream ended")
			return
		}
		_, _ = io.WriteString(w, "data: second\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	})
	f.upstreamHandler.Store(&handler)

	response := toolCall(t, f.url+"/mcp", "read_document", "d1", nil)
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("response = %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	events := bufio.NewReader(response.Body)
	readEvent := func(want string) {
		t.Helper()
		line, err := events.ReadString('\n')
		if err != nil || line != want+"\n" {
			t.Fatalf("event = %q, %v", line, err)
		}
		if _, err := events.ReadString('\n'); err != nil {
			t.Fatal(err)
		}
	}
	readEvent("data: first")
	close(firstRead)
	readEvent("data: second")

	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-canceled:
	case <-time.After(10 * time.Second):
		t.Fatal("the upstream request was not canceled")
	}
}
