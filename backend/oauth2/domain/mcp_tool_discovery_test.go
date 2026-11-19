package domain

import (
	"slices"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/protocol/mcp"
)

//spec:covers REQ-OAUTH2-095: メタデータは resource の path を継いだ URL、次に origin 直下の URL の順に探す。path が空なら一件だけである。
func TestProtectedResourceMetadataURLsInsertThePathFirst(t *testing.T) {
	cases := map[string][]string{
		"https://mcp.example.com/tenant/mcp": {
			"https://mcp.example.com/.well-known/oauth-protected-resource/tenant/mcp",
			"https://mcp.example.com/.well-known/oauth-protected-resource",
		},
		"https://mcp.example.com:8443": {"https://mcp.example.com:8443/.well-known/oauth-protected-resource"},
		"https://mcp.example.com/":     {"https://mcp.example.com/.well-known/oauth-protected-resource"},
	}
	for resource, want := range cases {
		got, err := ProtectedResourceMetadataURLs(resource)
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("%s: got %v %v; want %v", resource, got, err, want)
		}
	}
}

//spec:covers REQ-OAUTH2-095: resource が登録と文字列として一致し authorization_servers が文字列の配列のときだけ found とし、ほかは理由ごとの状態にする。
func TestParseProtectedResourceMetadataClassifiesTheDocument(t *testing.T) {
	const resource = "https://mcp.example.com/mcp"
	found, status := ParseProtectedResourceMetadata([]byte(`{
		"resource":"https://mcp.example.com/mcp",
		"authorization_servers":["https://idp.example.com/t1","https://other.example.com"],
		"scopes_supported":["files:read","files:write"],
		"dpop_signing_alg_values_supported":["ES256"],
		"unknown":{"ignored":true}
	}`), resource)
	if status != MetadataFound || found == nil {
		t.Fatalf("status=%s", status)
	}
	if !found.Trusts("https://idp.example.com/t1") || found.Trusts("https://idp.example.com/t2") {
		t.Fatalf("trust check: %+v", found)
	}
	if !slices.Equal(found.ScopesSupported, []string{"files:read", "files:write"}) || !found.SupportsES256DPoP() {
		t.Fatalf("metadata: %+v", found)
	}

	cases := map[string]MetadataStatus{
		`{"resource":"https://mcp.example.com/mcp/","authorization_servers":[]}`: MetadataResourceMismatch,
		`{"resource":"HTTPS://mcp.example.com/mcp","authorization_servers":[]}`:  MetadataResourceMismatch,
		`{"authorization_servers":[]}`:                                           MetadataResourceMismatch,
		`{"resource":"https://mcp.example.com/mcp"}`:                             MetadataInvalid,
		`{"resource":"https://mcp.example.com/mcp","authorization_servers":"x"}`: MetadataInvalid,
		`{"resource":"https://mcp.example.com/mcp","authorization_servers":[1]}`: MetadataInvalid,
		`["https://mcp.example.com/mcp"]`:                                        MetadataInvalid,
		`not json`:                                                               MetadataInvalid,
	}
	for raw, want := range cases {
		metadata, status := ParseProtectedResourceMetadata([]byte(raw), resource)
		if status != want || metadata != nil {
			t.Fatalf("%s: status=%s metadata=%+v; want %s", raw, status, metadata, want)
		}
	}

	withoutOptional, status := ParseProtectedResourceMetadata([]byte(`{"resource":"https://mcp.example.com/mcp","authorization_servers":["https://idp.example.com/t1"],"scopes_supported":"bad","dpop_signing_alg_values_supported":["PS256"]}`), resource)
	if status != MetadataFound || withoutOptional.ScopesSupported != nil || withoutOptional.SupportsES256DPoP() {
		t.Fatalf("optional members: status=%s %+v", status, withoutOptional)
	}
}

//spec:covers REQ-OAUTH2-095: 取得した一覧は REQ-MCPGATEWAY-011 と同じく切り詰め、100件を超えれば不完全にし、カタログの上限を満たさない名前を含めば全体を拒否する。
func TestCatalogToolsFromListedAppliesTheCatalogRules(t *testing.T) {
	arguments := make([]string, 40)
	for index := range arguments {
		arguments[index] = string(rune('a'+index%26)) + strings.Repeat("z", index/26)
	}
	slices.Sort(arguments)
	listed := []mcp.ListedTool{{Name: "search", Title: strings.Repeat("題", 300), Description: strings.Repeat("説", 3000), Arguments: arguments}}
	tools, complete, err := CatalogToolsFromListed(listed)
	if err != nil || !complete || len(tools) != 1 {
		t.Fatalf("tools=%+v complete=%v err=%v", tools, complete, err)
	}
	if got := tools[0]; len([]rune(got.Title)) != 256 || len([]rune(got.Description)) != 2048 || len(got.Arguments) != 32 || got.Arguments[0] != arguments[0] {
		t.Fatalf("truncation: %+v", got)
	}

	many := make([]mcp.ListedTool, 101)
	for index := range many {
		many[index] = mcp.ListedTool{Name: "tool_" + string(rune('a'+index%26)) + strings.Repeat("x", index/26)}
	}
	tools, complete, err = CatalogToolsFromListed(many)
	if err != nil || complete || len(tools) != 100 {
		t.Fatalf("over the limit: %d complete=%v err=%v", len(tools), complete, err)
	}

	for _, name := range []string{"", "a:b", "a#b", strings.Repeat("n", 129)} {
		if _, _, err := CatalogToolsFromListed([]mcp.ListedTool{{Name: name}}); err == nil {
			t.Fatalf("name %q accepted", name)
		}
	}
	if _, _, err := CatalogToolsFromListed([]mcp.ListedTool{{Name: "dup"}, {Name: "dup"}}); err == nil {
		t.Fatal("duplicate names accepted")
	}
}
