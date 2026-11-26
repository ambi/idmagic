package testing_mcptwin

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var (
	mcpExchanges = []string{
		"tools-list-first-page", "tools-list-last-page", "tools-list-empty-cursor", "tools-list-after-empty-cursor",
		"tools-list-unauthorized", "tools-list-insufficient-scope", "unsupported-protocol-version",
		"subscriptions-listen", "subscriptions-listen-without-tools",
	}
	// replayOnlyExchanges は模倣が作らない応答で、クライアントの再生にだけ使う。
	replayOnlyExchanges = []string{"subscriptions-listen-other-subscription"}
)

// 模倣は、記録した往復の要求に記録と同じ応答を返す。
func TestTwin_MatchesMCP20260728Recording(t *testing.T) {
	for _, diff := range Diverge(NewTwin(), MCP20260728Recording(t).Select(t, mcpExchanges...)) {
		t.Error(diff)
	}
}

// 模倣は、RFC 9728 §3.2 の例と同じ Protected Resource Metadata を返す。
func TestTwin_MatchesRFC9728Recording(t *testing.T) {
	for _, diff := range Diverge(NewTwin(), RFC9728Recording(t)) {
		t.Error(diff)
	}
}

// 記録のすべての往復が、照合か再生だけの一覧のどちらかに含まれる。往復を足したときに照合から漏れないためである。
func TestTwin_FidelityCoversEveryExchange(t *testing.T) {
	covered := map[string]bool{}
	for _, name := range append(append([]string{}, mcpExchanges...), replayOnlyExchanges...) {
		covered[name] = true
	}
	for _, ex := range MCP20260728Recording(t).Exchanges {
		if !covered[ex.Name] {
			t.Errorf("exchange %q is not compared with the twin", ex.Name)
		}
	}
}

// 照合が空虚でないことを確かめる。模倣の応答を記録から外すと、照合が食い違いを報告する。
func TestDiverge_ReportsTwinThatLeavesTheRecording(t *testing.T) {
	tests := []struct {
		name  string
		setup func(tw *Twin)
		want  string
	}{
		{"一ページ目が 500 を返す", func(tw *Twin) {
			tw.InjectFaults(Fault{Status: 500, Body: "boom"})
		}, "tools-list-first-page: status = 500, recorded 200"},
		{"二ページ目が不正な本文を返す", func(tw *Twin) {
			tw.InjectFaults(Fault{}, Fault{Status: 200, ContentType: "application/json", Body: "{"})
		}, "tools-list-last-page: body cannot be decoded"},
		{"受信確認が SSE でない", func(tw *Twin) {
			tw.InjectFaults(make([]Fault, 7)...)
			tw.InjectFaults(Fault{Status: 200, ContentType: "application/json", Body: `{"jsonrpc":"2.0","id":1,"result":{}}`})
		}, `subscriptions-listen: header Content-Type = "application/json", recorded "text/event-stream"`},
		{"401 が WWW-Authenticate を持たない", func(tw *Twin) {
			tw.InjectFaults(make([]Fault, 4)...)
			tw.InjectFaults(Fault{Status: 401})
		}, `tools-list-unauthorized: header WWW-Authenticate is missing`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tw := NewTwin()
			tc.setup(tw)
			diffs := Diverge(tw, MCP20260728Recording(t).Select(t, mcpExchanges...))
			if !strings.Contains(strings.Join(diffs, "\n"), tc.want) {
				t.Errorf("Diverge() = %q, want a line %q", diffs, tc.want)
			}
		})
	}
	t.Run("PRM の resource が違う", func(t *testing.T) {
		tw := NewTwin()
		rec := RFC9728Recording(t)
		rec.Exchanges[0].Setup = json.RawMessage(`{"resource":"https://resource.example.com","authorization_servers":["https://as1.example.com"]}`)
		diffs := Diverge(tw, rec)
		if want := "protected-resource-metadata: body.authorization_servers = [https://as1.example.com], recorded [https://as1.example.com https://as2.example.net]"; !strings.Contains(strings.Join(diffs, "\n"), want) {
			t.Errorf("Diverge() = %q, want a line %q", diffs, want)
		}
	})
}

// 再生は、記録にない値をクライアントが送ると食い違いを報告し、clientInfo の欠落だけは許す。
func TestCompareRequestBody(t *testing.T) {
	recorded := []byte(`{"jsonrpc":"2.0","method":"tools/list","params":{"cursor":"c","_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"ExampleClient"}}}}`)
	if diffs := CompareRequestBody(recorded, []byte(`{"jsonrpc":"2.0","id":7,"method":"tools/list","params":{"cursor":"c","_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`)); len(diffs) != 0 {
		t.Errorf("CompareRequestBody() = %q, want none", diffs)
	}
	diffs := CompareRequestBody(recorded, []byte(`{"jsonrpc":"2.0","method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2025-11-25"}}}`))
	for _, want := range []string{"body.params.cursor is missing", "body.params._meta.io.modelcontextprotocol/protocolVersion = 2025-11-25, recorded 2026-07-28"} {
		if !strings.Contains(strings.Join(diffs, "\n"), want) {
			t.Errorf("CompareRequestBody() = %q, want a line %q", diffs, want)
		}
	}
}

func listRequest(t *testing.T, url string, cursor *string) (int, map[string]any) {
	t.Helper()
	params := map[string]any{"_meta": map[string]any{"io.modelcontextprotocol/protocolVersion": RecordedProtocolVersion, "io.modelcontextprotocol/clientCapabilities": map[string]any{}}}
	if cursor != nil {
		params["cursor"] = *cursor
	}
	body, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": params})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, url, strings.NewReader(string(body)))
	req.Header.Set("Mcp-Protocol-Version", RecordedProtocolVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}

// 記録にない組み合わせ：空文字列のカーソルと欠落の区別、未知のカーソル、ツールの削除。
func TestTwin_PagesWithOpaqueCursors(t *testing.T) {
	tw := NewTwin()
	tw.PageSize, tw.Cursors = 1, []string{""}
	tw.AddTool(map[string]any{"name": "a"})
	tw.AddTool(map[string]any{"name": "b"})
	tw.AddTool(map[string]any{"name": "c"})
	server := httptest.NewServer(tw)
	defer server.Close()

	_, first := listRequest(t, server.URL, nil)
	if result := first["result"].(map[string]any); result["nextCursor"] != "" {
		t.Fatalf("first page = %v, want nextCursor \"\"", result)
	}
	empty, second := "", "page-2"
	_, page := listRequest(t, server.URL, &empty)
	if result := page["result"].(map[string]any); result["nextCursor"] != second {
		t.Fatalf("second page = %v, want nextCursor %q", result, second)
	}
	_, last := listRequest(t, server.URL, &second)
	if result := last["result"].(map[string]any); result["nextCursor"] != nil || result["tools"].([]any)[0].(map[string]any)["name"] != "c" {
		t.Fatalf("last page = %v, want c without nextCursor", result)
	}
	unknown := "x"
	if status, body := listRequest(t, server.URL, &unknown); status != 400 || body["error"].(map[string]any)["code"] != float64(-32602) {
		t.Errorf("unknown cursor = %d %v, want 400 and -32602", status, body)
	}
	tw.RemoveTool("b")
	_, page = listRequest(t, server.URL, &empty)
	if result := page["result"].(map[string]any); result["tools"].([]any)[0].(map[string]any)["name"] != "c" || result["nextCursor"] != nil {
		t.Errorf("second page after removing b = %v, want c without nextCursor", result)
	}
}

// 開いている購読は、ツールの追加と削除ごとに通知を受け、EndSubscriptions で成功の応答を受けて閉じる。
func TestTwin_NotifiesOpenSubscriptions(t *testing.T) {
	tw := NewTwin()
	server := httptest.NewServer(tw)
	defer server.Close()
	rec := MCP20260728Recording(t).Exchange(t, "subscriptions-listen")
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(string(rec.Request.Body)))
	req.Header.Set("Mcp-Protocol-Version", RecordedProtocolVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	events := bufio.NewReader(resp.Body)
	next := func() string {
		for {
			line, err := events.ReadString('\n')
			if err != nil {
				t.Fatalf("read stream: %v", err)
			}
			if data, ok := strings.CutPrefix(strings.TrimSpace(line), "data: "); ok {
				return data
			}
		}
	}
	if ack := next(); !strings.Contains(ack, `"toolsListChanged":true`) {
		t.Fatalf("first message = %s, want an acknowledgment with toolsListChanged", ack)
	}
	deadline := time.Now().Add(5 * time.Second)
	for tw.Subscribers() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	tw.AddTool(map[string]any{"name": "a"})
	if got := next(); !strings.Contains(got, "notifications/tools/list_changed") {
		t.Errorf("after AddTool = %s, want list_changed", got)
	}
	tw.RemoveTool("a")
	if got := next(); !strings.Contains(got, "notifications/tools/list_changed") {
		t.Errorf("after RemoveTool = %s, want list_changed", got)
	}
	tw.EndSubscriptions()
	if got := next(); !strings.Contains(got, `"id":1`) || !strings.Contains(got, `"result"`) {
		t.Errorf("after EndSubscriptions = %s, want the listen result", got)
	}
	if rest, _ := io.ReadAll(events); strings.TrimSpace(string(rest)) != "" {
		t.Errorf("stream continued after the result: %q", rest)
	}
}

// 版の _meta とヘッダーの食い違いは -32020、トークンのない要求は 401 になる。
func TestTwin_RejectsMismatchedVersionAndMissingToken(t *testing.T) {
	tw := NewTwin()
	server := httptest.NewServer(tw)
	defer server.Close()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{}}}}`
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, strings.NewReader(body))
	req.Header.Set("Mcp-Protocol-Version", "2025-11-25")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != 400 || !strings.Contains(string(raw), "-32020") {
		t.Errorf("header mismatch = %d %s, want 400 and -32020", resp.StatusCode, raw)
	}

	guarded := NewTwin()
	guarded.Tokens = map[string][]string{"t": {"files:read"}}
	guardedServer := httptest.NewServer(guarded)
	defer guardedServer.Close()
	if status, _ := listRequest(t, guardedServer.URL, nil); status != 401 {
		t.Errorf("without a token = %d, want 401", status)
	}
}
