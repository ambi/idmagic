package domain

import (
	"strings"
	"testing"
)

//spec:covers REQ-OAUTH2-064: ツール名は接続先内で一意にし、対象抽出の欠落や組み込み操作の不正な関係を拒否する。
func TestToolCatalogValidation(t *testing.T) {
	rule := func() ResourceAccessRule {
		return ResourceAccessRule{ResourceType: "idmagic_mcp_operation", Relation: "invoke", Scopes: []string{"read"}, Tool: &ResourceAccessTool{ID: "operation-" + "read", Name: "read", DisplayName: "閲覧"}}
	}
	cases := []struct {
		name   string
		change func(*ResourceAccessRule)
		valid  bool
	}{
		{name: "組み込み", valid: true},
		{name: "対象別", valid: true, change: func(r *ResourceAccessRule) {
			r.ResourceType = "document"
			r.Relation = "viewer"
			r.Tool.ResourceIDArgument = "document_id"
		}},
		{name: "引数欠落", change: func(r *ResourceAccessRule) { r.ResourceType = "document" }},
		{name: "関係不正", change: func(r *ResourceAccessRule) { r.Relation = "owner" }},
		{name: "余分な引数", change: func(r *ResourceAccessRule) { r.Tool.ResourceIDArgument = "id" }},
		{name: "長すぎるツール名", change: func(r *ResourceAccessRule) { r.Tool.Name = strings.Repeat("a", 129) }},
		{name: "区切り文字", change: func(r *ResourceAccessRule) { r.Tool.Name = "documents:read" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := rule()
			if tc.change != nil {
				tc.change(&r)
			}
			server := McpResourceServer{Scopes: []string{"read"}, AccessRules: []ResourceAccessRule{r}}
			if err := server.validateAccessRules(); (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
		})
	}
	first, second := rule(), rule()
	second.Tool.Name = "delete"
	server := McpResourceServer{Scopes: []string{"read"}, AccessRules: []ResourceAccessRule{first, second}}
	if err := server.validateAccessRules(); err != nil {
		t.Fatal(err)
	}
	second.Tool.Name = "read"
	server.AccessRules[1] = second
	if err := server.validateAccessRules(); err == nil {
		t.Fatal("duplicate tool accepted")
	}
	legacy := ResourceAccessRule{ResourceType: "document", Relation: "viewer", Scopes: []string{"read"}}
	tool := rule()
	tool.ResourceType = "document"
	tool.Relation = "viewer"
	tool.Tool.ResourceIDArgument = "id"
	for _, rules := range [][]ResourceAccessRule{{legacy, tool}, {tool, legacy}} {
		server.AccessRules = rules
		if err := server.validateAccessRules(); err == nil {
			t.Fatal("legacy and tool rule overlap accepted")
		}
	}
}
