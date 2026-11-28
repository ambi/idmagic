package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

//spec:covers REQ-MCPGATEWAY-015: 一覧の形式を問わず、許可名と元の定義、ページのカーソルを保ち、キャッシュを共有させない。
func TestFilterToolsListKeepsOnlyAllowedDefinitions(t *testing.T) {
	page := `{"jsonrpc":"2.0","id":7,"result":{"resultType":"complete","tools":[{"name":"read","description":"visible","inputSchema":{"type":"object"}},{"name":"write","description":"hidden"}],"nextCursor":"next","ttlMs":999,"cacheScope":"public"}}`
	for _, content := range []string{"application/json", "text/event-stream"} {
		t.Run(content, func(t *testing.T) {
			raw := page
			if content == "text/event-stream" {
				raw = ": keepalive\r\n\r\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\"params\":{\"message\":\"hidden progress\"}}\r\n\r\nevent: message\r\ndata: " + strings.Replace(page, `"result":`, "\r\ndata: \"result\":", 1) + "\r\n\r\n"
			}
			response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {content}, "Etag": {"public"}}, Body: io.NopCloser(strings.NewReader(raw))}
			if err := FilterToolsListResponse(response, json.RawMessage(`7`), []string{"read"}); err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(response.Body)
			if strings.Contains(string(body), "hidden") || strings.Contains(string(body), "write") || !strings.Contains(string(body), `"inputSchema":{"type":"object"}`) || !strings.Contains(string(body), `"nextCursor":"next"`) || !strings.Contains(string(body), `"cacheScope":"private"`) || !strings.Contains(string(body), `"ttlMs":0`) || response.Header.Get("Content-Type") != "application/json" || response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("ETag") != "" {
				t.Fatalf("unsafe list: %s %v", body, response.Header)
			}
		})
	}
}

//spec:covers REQ-MCPGATEWAY-015: 空のページでもカーソルを失わず、不正な上流の応答や上限超過を本文の公開前に拒否する。
func TestFilterToolsListRejectsUnverifiableResponses(t *testing.T) {
	for _, raw := range []string{
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"hidden"}],"nextCursor":""}}`,
	} {
		response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(raw))}
		if err := FilterToolsListResponse(response, json.RawMessage(`7`), []string{}); err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(response.Body)
		if !strings.Contains(string(body), `"tools":[]`) || (strings.Contains(raw, "nextCursor") && !strings.Contains(string(body), `"nextCursor":""`)) {
			t.Fatalf("lost empty page or cursor: %s", body)
		}
	}
	for _, raw := range []string{
		`{"jsonrpc":"2.0","id":8,"result":{"tools":[]}}`,
		`{"jsonrpc":"2.0","id":7,"result":{}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":null}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[],"Tools":[]}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"read"},{"name":"read"}]}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":""}]}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[{"Name":"read"}]}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[],"nextCursor":3}}`,
		`{"jsonrpc":"2.0","id":7,"error":{"code":-1}}`,
		`{"jsonrpc":"2.0","id":7,"result":{"tools":[]}} {}`,
		strings.Repeat(" ", (4<<20)+1),
	} {
		response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(raw))}
		if err := FilterToolsListResponse(response, json.RawMessage(`7`), []string{"read"}); err == nil {
			t.Fatalf("accepted unverifiable body: %.100s", raw)
		}
	}
}

//spec:covers REQ-MCPGATEWAY-015: 終了しない一覧の応答を期限で閉じ、未検査の断片を返さない。
func TestToolListReadBudgetClosesAnUnfinishedStream(t *testing.T) {
	reader, writer := io.Pipe()
	defer func() { _ = writer.Close() }()
	defer func() { _ = reader.Close() }()
	if raw, err := readToolsListBody(reader, time.Millisecond); err == nil || len(raw) != 0 {
		t.Fatalf("unfinished stream escaped its budget: %q %v", raw, err)
	}
}

//spec:covers REQ-MCPGATEWAY-015: 有効な一覧でも 4 MiB を超えた瞬間に拒否し、上限ちょうどの応答は受ける。
func TestFilteredToolsListSizeBoundary(t *testing.T) {
	page := `{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}`
	for _, size := range []int{4 << 20, (4 << 20) + 1} {
		raw := page + strings.Repeat(" ", size-len(page))
		response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(raw))}
		err := FilterToolsListResponse(response, json.RawMessage(`7`), []string{})
		if (err == nil) != (size == 4<<20) {
			t.Fatalf("size=%d err=%v", size, err)
		}
	}
}

//spec:covers REQ-MCPGATEWAY-015: 不正な SSE の結果、圧縮、異なる応答形式を素通ししない。
func TestFilteredToolsListRejectsUnsafeTransportResponses(t *testing.T) {
	page := `{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}`
	for _, tc := range []struct {
		status                  int
		content, encoding, body string
	}{
		{200, "text/event-stream", "", "data: " + page + "\n\ndata: " + page + "\n\n"},
		{200, "text/event-stream", "", ": keepalive\n\n"},
		{200, "text/event-stream", "", "data: {\"jsonrpc\":\"2.0\",\"id\":8,\"result\":{\"tools\":[]}}\n\n"},
		{200, "application/json", "gzip", page},
		{200, "text/html", "", page},
		{302, "application/json", "", page},
	} {
		response := &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {tc.content}}, Body: io.NopCloser(strings.NewReader(tc.body))}
		response.Header.Set("Content-Encoding", tc.encoding)
		if err := FilterToolsListResponse(response, json.RawMessage(`7`), []string{}); err == nil {
			t.Fatalf("accepted unsafe response: %+v", tc)
		}
	}
}

//spec:covers REQ-MCPGATEWAY-015: 任意の応答を受理した場合も、結果の tools に許可外の名前を残さない。
func FuzzToolsListFiltering(f *testing.F) {
	f.Add([]byte(`{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"read"},{"name":"hidden"}]}}`), false)
	f.Add([]byte("data: {\"jsonrpc\":\"2.0\",\"id\":7,\"result\":{\"tools\":[{\"name\":\"read\"}]}}\n\n"), true)
	f.Fuzz(func(t *testing.T, raw []byte, sse bool) {
		if len(raw) > 64<<10 {
			t.Skip()
		}
		content := "application/json"
		if sse {
			content = "text/event-stream"
		}
		response := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {content}}, Body: io.NopCloser(strings.NewReader(string(raw)))}
		if err := FilterToolsListResponse(response, json.RawMessage(`7`), []string{"read"}); err != nil {
			return
		}
		var result struct {
			Result struct{ Tools []struct{ Name string } }
		}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		for _, tool := range result.Result.Tools {
			if tool.Name != "read" {
				t.Fatalf("unassigned tool: %q", tool.Name)
			}
		}
	})
}
