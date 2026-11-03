package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func reportJSON(t *testing.T, tools []map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(tools)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func repeatedTools(count int) []map[string]any {
	tools := make([]map[string]any, count)
	for index := range tools {
		tools[index] = map[string]any{"name": "tool_" + strings.Repeat("x", index%3) + string(rune('a'+index%26)) + string(rune('a'+index/26))}
	}
	return tools
}

//spec:covers REQ-OAUTH2-086: 報告の一覧は件数と各項目の上限の内側だけを受け入れる。
func TestParseReportedToolsAcceptsTheLimits(t *testing.T) {
	arguments := make([]string, 32)
	for index := range arguments {
		arguments[index] = string(rune('a'+index%26)) + strings.Repeat("b", index/26)
	}
	if tools, err := ParseReportedTools(reportJSON(t, []map[string]any{{"name": "a", "arguments": arguments}})); err != nil || len(tools[0].Arguments) != 32 {
		t.Fatalf("32 arguments: %+v %v", tools, err)
	}
	accepted := []map[string]any{{
		"name": strings.Repeat("n", 128), "title": strings.Repeat("題", 256), "description": strings.Repeat("説", 2048),
		"arguments": []string{strings.Repeat("a", 128), "document_id"},
	}}
	tools, err := ParseReportedTools(reportJSON(t, accepted))
	if err != nil || len(tools) != 1 || tools[0].Arguments[1] != "document_id" {
		t.Fatalf("tools=%+v err=%v", tools, err)
	}
	if tools, err := ParseReportedTools(reportJSON(t, repeatedTools(100))); err != nil || len(tools) != 100 {
		t.Fatalf("100 tools: len=%d err=%v", len(tools), err)
	}
	if tools, err := ParseReportedTools("[]"); err != nil || len(tools) != 0 {
		t.Fatalf("empty report: %v %v", tools, err)
	}
}

//spec:covers REQ-OAUTH2-086: 上限を一つでも越える報告と、形の誤った報告を拒否する。
func TestParseReportedToolsRejectsOutOfRangeReports(t *testing.T) {
	manyArguments := make([]string, 33)
	for index := range manyArguments {
		manyArguments[index] = string(rune('a' + index%26))
	}
	cases := map[string]string{
		"not an array":        `{"name":"a"}`,
		"not json":            `[`,
		"101 tools":           reportJSON(t, repeatedTools(101)),
		"empty name":          reportJSON(t, []map[string]any{{"name": ""}}),
		"129 rune name":       reportJSON(t, []map[string]any{{"name": strings.Repeat("n", 129)}}),
		"colon in name":       reportJSON(t, []map[string]any{{"name": "a:b"}}),
		"hash in name":        reportJSON(t, []map[string]any{{"name": "a#b"}}),
		"duplicate name":      reportJSON(t, []map[string]any{{"name": "a"}, {"name": "a"}}),
		"257 rune title":      reportJSON(t, []map[string]any{{"name": "a", "title": strings.Repeat("t", 257)}}),
		"2049 rune text":      reportJSON(t, []map[string]any{{"name": "a", "description": strings.Repeat("d", 2049)}}),
		"33 arguments":        reportJSON(t, []map[string]any{{"name": "a", "arguments": manyArguments}}),
		"129 rune argument":   reportJSON(t, []map[string]any{{"name": "a", "arguments": []string{strings.Repeat("a", 129)}}}),
		"non-string argument": `[{"name":"a","arguments":[1]}]`,
	}
	for name, raw := range cases {
		if tools, err := ParseReportedTools(raw); err == nil {
			t.Errorf("%s: accepted %+v", name, tools)
		}
	}
}
