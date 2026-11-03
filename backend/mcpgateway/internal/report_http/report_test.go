package report_http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"
)

type received struct {
	tools    []Tool
	complete string
	user     string
	password string
}

// newServers は、ページごとの応答を返す上流と、報告を受ける IdMagic の代役を立てる。
func newServers(t *testing.T, pages func(cursor string) (int, string)) (Reporter, *received, *atomic.Int32) {
	t.Helper()
	got := &received{}
	reports := &atomic.Int32{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Params struct {
				Cursor string `json:"cursor"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		status, body := pages(request.Params.Cursor)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(upstream.Close)
	idmagic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reports.Add(1)
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		got.user, got.password, _ = r.BasicAuth()
		got.complete = r.PostForm.Get("complete")
		if err := json.Unmarshal([]byte(r.PostForm.Get("tools")), &got.tools); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(idmagic.Close)
	return Reporter{
		Upstream: upstream.URL, ReportURL: idmagic.URL, ClientID: "gateway", ClientSecret: "secret",
		UpstreamClient: upstream.Client(), IdMagicClient: idmagic.Client(),
	}, got, reports
}

func page(t *testing.T, tools []map[string]any, next string) string {
	t.Helper()
	result := map[string]any{"tools": tools}
	if next != "" {
		result["nextCursor"] = next
	}
	raw, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

//spec:covers REQ-MCPGATEWAY-011: nextCursor をたどって集め、送る項目は名前、表示名、説明、引数のキーだけで、上限で切り詰める。
func TestReportFollowsCursorsAndSendsOnlyTheDeclaredFields(t *testing.T) {
	long := strings.Repeat("説", 3000)
	reporter, got, _ := newServers(t, func(cursor string) (int, string) {
		if cursor == "" {
			return http.StatusOK, page(t, []map[string]any{{
				"name": "read_document", "title": strings.Repeat("題", 300), "description": long,
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{"document_id": map[string]any{"type": "string"}}},
				"annotations": map[string]any{"readOnlyHint": true},
			}}, "next")
		}
		// SSE の data 行でも読める。
		return http.StatusOK, "event: message\ndata: " + page(t, []map[string]any{{"name": "delete_document"}}, "") + "\n\n"
	})
	result, err := reporter.Report(context.Background())
	if err != nil || result.Tools != 2 || !result.Complete {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if got.complete != "true" || got.user != "gateway" || got.password != "secret" || len(got.tools) != 2 {
		t.Fatalf("received=%+v", got)
	}
	first := got.tools[0]
	if first.Name != "read_document" || utf8.RuneCountInString(first.Title) != maxTitle || utf8.RuneCountInString(first.Description) != maxDescription || len(first.Arguments) != 1 || first.Arguments[0] != "document_id" {
		t.Fatalf("first tool=%+v", first)
	}
}

//spec:covers REQ-MCPGATEWAY-011: 二ページ目以降の失敗、件数とページの上限では、集めた分を不完全として報告する。
func TestReportMarksPartialListsIncomplete(t *testing.T) {
	cases := map[string]struct {
		pages     func(cursor string) (int, string)
		wantTools int
	}{
		"second page fails": {func(cursor string) (int, string) {
			if cursor == "" {
				return http.StatusOK, page(t, []map[string]any{{"name": "a"}}, "next")
			}
			return http.StatusBadGateway, ""
		}, 1},
		"more than 100 tools": {func(string) (int, string) {
			tools := make([]map[string]any, 101)
			for index := range tools {
				tools[index] = map[string]any{"name": fmt.Sprintf("tool_%d", index)}
			}
			return http.StatusOK, page(t, tools, "")
		}, maxTools},
		"more than 10 pages": {func(cursor string) (int, string) {
			return http.StatusOK, page(t, []map[string]any{{"name": "tool_" + url.QueryEscape(cursor)}}, cursor+"x")
		}, maxPages},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			reporter, got, _ := newServers(t, tc.pages)
			result, err := reporter.Report(context.Background())
			if err != nil || result.Complete || result.Tools != tc.wantTools || got.complete != "false" || len(got.tools) != tc.wantTools {
				t.Fatalf("result=%+v err=%v received=%s/%d", result, err, got.complete, len(got.tools))
			}
		})
	}
}

//spec:covers REQ-MCPGATEWAY-011: 最初のページを取得できなければ報告しない。
func TestReportSendsNothingWhenTheFirstPageFails(t *testing.T) {
	for name, pages := range map[string]func(string) (int, string){
		"HTTP error":     func(string) (int, string) { return http.StatusUnauthorized, "" },
		"JSON-RPC error": func(string) (int, string) { return http.StatusOK, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601}}` },
		"not JSON":       func(string) (int, string) { return http.StatusOK, "<html>" },
	} {
		t.Run(name, func(t *testing.T) {
			reporter, _, reports := newServers(t, pages)
			if _, err := reporter.Report(context.Background()); err == nil {
				t.Fatal("reported a failed listing")
			}
			if reports.Load() != 0 {
				t.Fatalf("reports=%d, want none", reports.Load())
			}
		})
	}
}
