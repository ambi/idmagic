package mcp

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestParseToolsListPageReadsJSONAndTheLastSSEMessage(t *testing.T) {
	page := `{"jsonrpc":"2.0","id":1,"result":{"nextCursor":"c2","tools":[` +
		`{"name":"search","title":"Search","description":"Find documents","inputSchema":{"type":"object","properties":{"query":{},"limit":{}}}},` +
		`{"name":"ping"}]}}`
	for name, raw := range map[string]string{
		"json": page,
		"sse":  "event: message\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\"}\n\ndata: " + page + "\n\n",
	} {
		tools, next, err := ParseToolsListPage([]byte(raw))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if (next == nil || *next != "c2") || len(tools) != 2 {
			t.Fatalf("%s: tools=%+v next=%v", name, tools, next)
		}
		if got := tools[0]; got.Name != "search" || got.Title != "Search" || got.Description != "Find documents" || !slices.Equal(got.Arguments, []string{"limit", "query"}) {
			t.Fatalf("%s: first tool %+v; want sorted argument names", name, got)
		}
		if tools[1].Name != "ping" || len(tools[1].Arguments) != 0 {
			t.Fatalf("%s: second tool %+v", name, tools[1])
		}
	}
}

// 空文字列の nextCursor は有効なカーソルで、欠落と null だけが一覧の終わりである（MCP 2026-07-28 の server/utilities/pagination）。
func TestParseToolsListPageDistinguishesAnEmptyCursorFromTheEnd(t *testing.T) {
	for name, tc := range map[string]struct {
		raw  string
		want *string
	}{
		"empty":   {`{"jsonrpc":"2.0","id":1,"result":{"tools":[],"nextCursor":""}}`, new("")},
		"missing": {`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`, nil},
		"null":    {`{"jsonrpc":"2.0","id":1,"result":{"tools":[],"nextCursor":null}}`, nil},
	} {
		_, next, err := ParseToolsListPage([]byte(tc.raw))
		if err != nil || (next == nil) != (tc.want == nil) || (next != nil && *next != *tc.want) {
			t.Errorf("%s: next=%v err=%v, want %v", name, next, err, tc.want)
		}
	}
}

func TestParseToolsListPageRejectsErrorsAndNonResults(t *testing.T) {
	for name, raw := range map[string]string{
		"error":           `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"no"}}`,
		"no result":       `{"jsonrpc":"2.0","id":1}`,
		"not json":        `<html></html>`,
		"tools not array": `{"jsonrpc":"2.0","id":1,"result":{"tools":{}}}`,
	} {
		if _, _, err := ParseToolsListPage([]byte(raw)); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}
}

func TestToolsListRequestCarriesTheProtocolVersionAndCursor(t *testing.T) {
	raw, err := ToolsListRequest(new("next"))
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Method string `json:"method"`
		Params struct {
			Cursor string         `json:"cursor"`
			Meta   map[string]any `json:"_meta"`
		} `json:"params"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body.Method != "tools/list" || body.Params.Cursor != "next" || body.Params.Meta[protocolVersionMetaKey] != ProtocolVersion {
		t.Fatalf("body = %s", raw)
	}
	first, err := ToolsListRequest(nil)
	if err != nil {
		t.Fatal(err)
	}
	var withoutCursor struct {
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(first, &withoutCursor); err != nil {
		t.Fatal(err)
	}
	if _, ok := withoutCursor.Params["cursor"]; ok {
		t.Fatalf("first page sends a cursor: %s", first)
	}
	empty, err := ToolsListRequest(new(""))
	if err != nil {
		t.Fatal(err)
	}
	var withEmptyCursor struct {
		Params map[string]any `json:"params"`
	}
	if err := json.Unmarshal(empty, &withEmptyCursor); err != nil {
		t.Fatal(err)
	}
	if cursor, ok := withEmptyCursor.Params["cursor"]; !ok || cursor != "" {
		t.Fatalf("an empty cursor is not sent: %s", empty)
	}
}
