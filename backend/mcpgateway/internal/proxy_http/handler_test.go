package proxy_http

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
	"github.com/ambi/idmagic/backend/mcpgateway/internal/evaluation_http"
)

const resourceURL = "https://mcp.example.test/mcp"

type stubEvaluator struct {
	result evaluation_http.Result
	err    error
	calls  atomic.Int32
	last   evaluation_http.Request
}

func (s *stubEvaluator) Evaluate(_ context.Context, in evaluation_http.Request) (evaluation_http.Result, error) {
	s.calls.Add(1)
	s.last = in
	return s.result, s.err
}

type harness struct {
	gateway   *httptest.Server
	evaluator *stubEvaluator
	calls     *atomic.Int32
	received  *atomic.Value
	audit     *bytes.Buffer
	metrics   *Metrics
}

func newHarness(t *testing.T, upstream http.HandlerFunc, change func(*Options)) *harness {
	t.Helper()
	h := &harness{
		evaluator: &stubEvaluator{result: evaluation_http.Result{Available: true, Permitted: true, RequestID: "eval-1"}},
		calls:     &atomic.Int32{}, received: &atomic.Value{}, audit: &bytes.Buffer{}, metrics: NewMetrics(),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.calls.Add(1)
		h.received.Store(r.Clone(r.Context()))
		upstream(w, r)
	}))
	t.Cleanup(server.Close)
	upstreamURL, _ := url.Parse(server.URL + "/internal")
	resource, _ := url.Parse(resourceURL)
	options := Options{
		Resource: resource, Upstream: upstreamURL, RealmURL: "https://id.example.test/realms/default",
		MaxBodyBytes: 4096,
		Mapping: domain.Mapping{
			Methods: map[string]domain.OperationRule{"tools/list": {ResourceType: "mcp_server", Relation: "list", ResourceID: "main"}},
			Tools:   map[string]domain.OperationRule{"read": {ResourceType: "document", Relation: "viewer", ResourceID: "d1"}},
		},
		Evaluator: h.evaluator, Audit: slog.New(slog.NewJSONHandler(h.audit, nil)), Metrics: h.metrics,
		Transport: http.DefaultTransport,
	}
	if change != nil {
		change(&options)
	}
	handler, err := NewHandler(options)
	if err != nil {
		t.Fatal(err)
	}
	h.gateway = httptest.NewServer(handler)
	t.Cleanup(h.gateway.Close)
	return h
}

func toolBody(name string) string {
	return `{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"` + name +
		`","_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
}

func toolRequest(t *testing.T, target, name string) *http.Request {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, target+"/mcp", strings.NewReader(toolBody(name)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer secret-token")
	request.Header.Set("Mcp-Protocol-Version", domain.ProtocolVersion)
	request.Header.Set("Mcp-Method", domain.MethodToolsCall)
	request.Header.Set("Mcp-Name", name)
	return request
}

func do(t *testing.T, request *http.Request) (*http.Response, map[string]any) {
	t.Helper()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, _ := io.ReadAll(response.Body)
	body := map[string]any{}
	_ = json.Unmarshal(raw, &body)
	return response, body
}

func errorCode(body map[string]any) float64 {
	failure, _ := body["error"].(map[string]any)
	code, _ := failure["code"].(float64)
	return code
}

func accepted(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{}`) }

//spec:covers REQ-MCPGATEWAY-004: 本文を変えず、資格情報、クエリ、クライアントの相関 ID を上流へ渡さない
func TestForwardDropsCredentialsAndKeepsTheBody(t *testing.T) {
	var forwarded atomic.Value
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		forwarded.Store(string(raw))
		w.Header().Set("X-Request-ID", "from-upstream")
		w.Header().Set("X-Upstream", "kept")
		w.WriteHeader(http.StatusAccepted)
	}, nil)
	request := toolRequest(t, h.gateway.URL, "read")
	request.URL.RawQuery = "x=1"
	request.Header.Set("DPoP", "proof")
	request.Header.Set("Cookie", "a=b")
	request.Header.Set("X-Request-ID", "from-client")
	response, _ := do(t, request)

	if response.StatusCode != http.StatusAccepted || response.Header.Get("X-Upstream") != "kept" {
		t.Fatalf("status=%d headers=%v", response.StatusCode, response.Header)
	}
	got := h.received.Load().(*http.Request)
	for _, name := range []string{"Authorization", "DPoP", "Cookie"} {
		if got.Header.Get(name) != "" {
			t.Errorf("%s reached the upstream", name)
		}
	}
	if got.URL.Path != "/internal" || got.URL.RawQuery != "" {
		t.Errorf("upstream URL = %s", got.URL)
	}
	correlation := response.Header.Get("X-Request-ID")
	if correlation == "" || correlation == "from-client" || correlation == "from-upstream" || got.Header.Get("X-Request-ID") != correlation {
		t.Errorf("correlation: response=%q upstream=%q", correlation, got.Header.Get("X-Request-ID"))
	}
	if h.evaluator.last.CorrelationID != correlation {
		t.Errorf("evaluation correlation = %q", h.evaluator.last.CorrelationID)
	}
	if forwarded.Load() != toolBody("read") {
		t.Errorf("body changed: %v", forwarded.Load())
	}
}

// 評価の前にも評価の失敗でも、上流を呼ばず、理由を返さない。
//
//spec:covers REQ-MCPGATEWAY-002, REQ-MCPGATEWAY-003, REQ-MCPGATEWAY-005: 要求の形、不許可、評価不能の応答と、上流と評価の呼び出し回数
func TestNothingReachesTheUpstreamWithoutAPermit(t *testing.T) {
	cases := []struct {
		name        string
		result      evaluation_http.Result
		err         error
		change      func(*http.Request)
		status      int
		code        float64
		evaluations int32
		outcome     string
	}{
		{name: "denied", result: evaluation_http.Result{Available: true, Reasons: []string{"relation_denied"}}, status: 403, code: -32000, evaluations: 1, outcome: outcomeDenied},
		{name: "token rejected", result: evaluation_http.Result{Available: true, Reasons: []string{"token_rejected"}}, status: 401, code: -32001, evaluations: 1, outcome: outcomeUnauthenticated},
		{name: "sender proof", result: evaluation_http.Result{Available: true, Reasons: []string{"sender_proof_invalid"}}, status: 401, code: -32001, evaluations: 1, outcome: outcomeUnauthenticated},
		{name: "evaluation failed", err: errors.New("down"), status: 503, code: -32002, evaluations: 1, outcome: outcomeUnavailable},
		{name: "no decision", result: evaluation_http.Result{}, status: 503, code: -32002, evaluations: 1, outcome: outcomeUnavailable},
		{name: "no token", change: func(r *http.Request) { r.Header.Del("Authorization") }, status: 401, code: -32001, outcome: outcomeUnauthenticated},
		{name: "basic scheme", change: func(r *http.Request) { r.Header.Set("Authorization", "Basic abc") }, status: 401, code: -32001, outcome: outcomeUnauthenticated},
		{name: "two tokens", change: func(r *http.Request) { r.Header.Add("Authorization", "Bearer other") }, status: 401, code: -32001, outcome: outcomeUnauthenticated},
		{name: "two proofs", change: func(r *http.Request) { r.Header.Add("DPoP", "a"); r.Header.Add("DPoP", "b") }, status: 401, code: -32001, outcome: outcomeUnauthenticated},
		{name: "bad origin", change: func(r *http.Request) { r.Header.Set("Origin", "https://evil.example") }, status: 403, outcome: outcomeInvalid},
		{name: "gzip", change: func(r *http.Request) { r.Header.Set("Content-Encoding", "gzip") }, status: 415, outcome: outcomeInvalid},
		{name: "not json", change: func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, status: 415, outcome: outcomeInvalid},
		{name: "other path", change: func(r *http.Request) { r.URL.Path = "/mcp/other" }, status: 404, outcome: outcomeInvalid},
		{name: "get", change: func(r *http.Request) { r.Method = http.MethodGet }, status: 405, outcome: outcomeInvalid},
		{name: "delete", change: func(r *http.Request) { r.Method = http.MethodDelete }, status: 405, outcome: outcomeInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t, accepted, nil)
			h.evaluator.result, h.evaluator.err = tc.result, tc.err
			request := toolRequest(t, h.gateway.URL, "read")
			if tc.change != nil {
				tc.change(request)
			}
			response, body := do(t, request)
			if response.StatusCode != tc.status || errorCode(body) != tc.code {
				t.Fatalf("status=%d body=%v", response.StatusCode, body)
			}
			if h.calls.Load() != 0 {
				t.Fatalf("upstream called %d times", h.calls.Load())
			}
			if h.evaluator.calls.Load() != tc.evaluations {
				t.Fatalf("evaluated %d times", h.evaluator.calls.Load())
			}
			if strings.Contains(strings.ToLower(body2string(body)), "reason") || strings.Contains(body2string(body), "relation_denied") {
				t.Fatalf("reason leaked: %v", body)
			}
			if !strings.Contains(h.audit.String(), `"outcome":"`+tc.outcome+`"`) {
				t.Fatalf("audit = %s", h.audit)
			}
			if strings.Contains(h.audit.String(), "secret-token") {
				t.Fatal("token logged")
			}
		})
	}
}

func body2string(body map[string]any) string {
	raw, err := json.Marshal(body)
	if err != nil {
		return err.Error()
	}
	return string(raw)
}

//spec:covers REQ-MCPGATEWAY-003: 評価が許可でも宣言のないツールを転送しない
func TestUnmappedToolIsNeverForwardedEvenWhenPermitted(t *testing.T) {
	h := newHarness(t, accepted, nil)
	response, body := do(t, toolRequest(t, h.gateway.URL, "drop_table"))
	if response.StatusCode != http.StatusForbidden || errorCode(body) != -32000 || h.calls.Load() != 0 {
		t.Fatalf("status=%d body=%v upstream=%d", response.StatusCode, body, h.calls.Load())
	}
	if h.evaluator.calls.Load() != 1 || h.evaluator.last.Operation.ResourceID != "drop_table" {
		t.Fatalf("evaluation = %+v", h.evaluator.last)
	}
	if got := testutil.ToFloat64(h.metrics.requests.WithLabelValues("tools/call", domain.UnmappedTool, outcomeDenied)); got != 1 {
		t.Fatalf("metric = %v", got)
	}
}

//spec:covers REQ-MCPGATEWAY-003, REQ-OAUTH2-062: 429 の Retry-After を 503 へ写し、上流を呼ばない
func TestRateLimitedEvaluationKeepsRetryAfter(t *testing.T) {
	h := newHarness(t, accepted, nil)
	h.evaluator.result, h.evaluator.err = evaluation_http.Result{RetryAfter: "7", RequestID: "eval-2"}, errors.New("status 429")
	response, _ := do(t, toolRequest(t, h.gateway.URL, "read"))
	if response.StatusCode != http.StatusServiceUnavailable || response.Header.Get("Retry-After") != "7" || h.calls.Load() != 0 {
		t.Fatalf("status=%d retry=%q upstream=%d", response.StatusCode, response.Header.Get("Retry-After"), h.calls.Load())
	}
	if !strings.Contains(h.audit.String(), `"evaluation_request_id":"eval-2"`) {
		t.Fatalf("audit = %s", h.audit)
	}
}

//spec:covers REQ-MCPGATEWAY-002: 本文の上限は 413、未知の方法は 404 と -32601、古い版は -32022 と supported
func TestProtocolRejections(t *testing.T) {
	h := newHarness(t, accepted, func(o *Options) { o.MaxBodyBytes = 300 })
	large := toolRequest(t, h.gateway.URL, "read")
	large.Body = io.NopCloser(strings.NewReader(strings.Repeat(" ", 400) + toolBody("read")))
	large.ContentLength = -1
	if response, _ := do(t, large); response.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("large: status=%d", response.StatusCode)
	}

	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
	request, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, h.gateway.URL+"/mcp", strings.NewReader(initialize))
	request.Header = toolRequest(t, h.gateway.URL, "read").Header.Clone()
	request.Header.Set("Mcp-Method", "initialize")
	response, body := do(t, request)
	if response.StatusCode != http.StatusNotFound || errorCode(body) != domain.CodeMethodNotFound {
		t.Fatalf("initialize: status=%d body=%v", response.StatusCode, body)
	}
	if h.evaluator.calls.Load() != 0 || h.calls.Load() != 0 {
		t.Fatalf("evaluated=%d upstream=%d", h.evaluator.calls.Load(), h.calls.Load())
	}

	older := strings.ReplaceAll(toolBody("read"), "2026-07-28", "2025-11-25")
	request, _ = http.NewRequestWithContext(t.Context(), http.MethodPost, h.gateway.URL+"/mcp", strings.NewReader(older))
	request.Header = toolRequest(t, h.gateway.URL, "read").Header.Clone()
	request.Header.Set("Mcp-Protocol-Version", "2025-11-25")
	response, body = do(t, request)
	failure, _ := body["error"].(map[string]any)
	if response.StatusCode != http.StatusBadRequest || errorCode(body) != domain.CodeUnsupportedVersion || !strings.Contains(body2string(failure), `"supported":["2026-07-28"]`) {
		t.Fatalf("older: status=%d body=%v", response.StatusCode, body)
	}
}

// 最終の応答を待たずに、途中のイベントがクライアントへ届く。
//
//spec:covers REQ-MCPGATEWAY-004: text/event-stream をためずに書き出す
func TestStreamingEventsArriveBeforeTheStreamEnds(t *testing.T) {
	release := make(chan struct{})
	h := newHarness(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: progress\n\n")
		w.(http.Flusher).Flush()
		<-release
		_, _ = io.WriteString(w, "data: final\n\n")
	}, nil)
	defer close(release)
	response, err := http.DefaultClient.Do(toolRequest(t, h.gateway.URL, "read"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	line := make(chan string, 1)
	go func() {
		text, _ := bufio.NewReader(response.Body).ReadString('\n')
		line <- text
	}()
	select {
	case text := <-line:
		if text != "data: progress\n" {
			t.Fatalf("first line = %q", text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the event was buffered")
	}
}

//spec:covers REQ-MCPGATEWAY-004: クライアントの切断で上流の要求を取り消す
func TestClientDisconnectCancelsTheUpstream(t *testing.T) {
	cancelled := make(chan struct{})
	started := make(chan struct{})
	h := newHarness(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}, nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	response, err := http.DefaultClient.Do(toolRequest(t, h.gateway.URL, "read").WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	<-started
	cancel()
	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("the upstream was not cancelled")
	}
}

//spec:covers REQ-MCPGATEWAY-004: 上流へ接続できないときは 502 と -32003 で、再送しない
func TestUpstreamFailureIsReported(t *testing.T) {
	var attempts atomic.Int32
	h := newHarness(t, accepted, func(o *Options) {
		o.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return nil, errors.New("refused")
		})
	})
	response, body := do(t, toolRequest(t, h.gateway.URL, "read"))
	if response.StatusCode != http.StatusBadGateway || errorCode(body) != -32003 || attempts.Load() != 1 {
		t.Fatalf("status=%d body=%v attempts=%d", response.StatusCode, body, attempts.Load())
	}
	if !strings.Contains(h.audit.String(), `"outcome":"upstream_failed"`) {
		t.Fatalf("audit = %s", h.audit)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

//spec:covers REQ-MCPGATEWAY-005: 経路つきの保護リソースのメタデータと 401 の resource_metadata の一致、評価を呼ばないこと
func TestMetadataNamesTheResourceAndRealm(t *testing.T) {
	h := newHarness(t, accepted, nil)
	for _, path := range []string{"/.well-known/oauth-protected-resource/mcp", "/.well-known/oauth-protected-resource"} {
		request, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, h.gateway.URL+path, http.NoBody)
		response, body := do(t, request)
		if response.StatusCode != http.StatusOK || body["resource"] != resourceURL {
			t.Fatalf("%s: status=%d body=%v", path, response.StatusCode, body)
		}
		servers, _ := body["authorization_servers"].([]any)
		if len(servers) != 1 || servers[0] != "https://id.example.test/realms/default" {
			t.Fatalf("authorization_servers = %v", servers)
		}
	}
	if h.evaluator.calls.Load() != 0 {
		t.Fatal("metadata was evaluated")
	}
	request := toolRequest(t, h.gateway.URL, "read")
	request.Header.Del("Authorization")
	response, _ := do(t, request)
	want := `resource_metadata="https://mcp.example.test/.well-known/oauth-protected-resource/mcp"`
	if !strings.Contains(response.Header.Get("WWW-Authenticate"), want) {
		t.Fatalf("challenge = %q", response.Header.Get("WWW-Authenticate"))
	}
}

//spec:covers REQ-MCPGATEWAY-003: DPoP の証明と公開 URL の htu をそのまま評価へ渡す
func TestEvaluationReceivesTheProofAgainstThePublicURL(t *testing.T) {
	h := newHarness(t, accepted, nil)
	request := toolRequest(t, h.gateway.URL, "read")
	request.Header.Set("Authorization", "DPoP bound-token")
	request.Header.Set("DPoP", "the-proof")
	if response, _ := do(t, request); response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	last := h.evaluator.last
	if last.Token != "bound-token" || last.DPoPProof != "the-proof" || last.HTU != resourceURL {
		t.Fatalf("evaluation = %+v", last)
	}
}

//spec:covers REQ-MCPGATEWAY-006: 呼び出し元が決める長い値をログへそのまま書かない
func TestAuditBoundsCallerControlledValues(t *testing.T) {
	h := newHarness(t, accepted, nil)
	method := strings.Repeat("m", 1000)
	request, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, h.gateway.URL+"/mcp", strings.NewReader(
		`{"jsonrpc":"2.0","id":1,"method":"`+method+`","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`))
	request.Header = toolRequest(t, h.gateway.URL, "read").Header.Clone()
	request.Header.Set("Mcp-Method", method)
	if response, _ := do(t, request); response.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", response.StatusCode)
	}
	if strings.Contains(h.audit.String(), strings.Repeat("m", 129)) || !strings.Contains(h.audit.String(), strings.Repeat("m", 128)) {
		t.Fatalf("audit = %s", h.audit)
	}
}
