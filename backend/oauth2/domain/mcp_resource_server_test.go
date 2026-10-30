package domain

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func validMcpResourceServer() McpResourceServer {
	now := time.Now().UTC()
	return McpResourceServer{
		TenantID:  "tenant-1",
		ID:        "11111111-1111-1111-1111-111111111111",
		Resource:  "https://mcp.example.com/tools/github",
		Name:      "GitHub MCP Tools",
		Scopes:    []string{"mcp.read", "mcp.write"},
		State:     McpResourceServerActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestMcpResourceServerValidate_valid(t *testing.T) {
	m := validMcpResourceServer()
	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid McpResourceServer, got error: %v", err)
	}
}

func TestMcpResourceServerValidate_rejectsEmptyID(t *testing.T) {
	m := validMcpResourceServer()
	m.ID = ""
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for empty resource_server_id")
	}
}

func TestMcpResourceServerValidate_rejectsNonURLResource(t *testing.T) {
	m := validMcpResourceServer()
	m.Resource = "not a url"
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for invalid resource URI")
	}
}

func TestMcpResourceServerValidate_rejectsResourceWithFragment(t *testing.T) {
	m := validMcpResourceServer()
	m.Resource = "https://mcp.example.com/tools/github#section"
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for resource URI containing a fragment (RFC 8707 §2)")
	}
}

func TestMcpResourceServerValidate_rejectsEmptyScopes(t *testing.T) {
	m := validMcpResourceServer()
	m.Scopes = nil
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for empty scopes")
	}
}

func TestMcpResourceServerValidate_rejectsUnknownState(t *testing.T) {
	m := validMcpResourceServer()
	m.State = "Bogus"
	if err := m.Validate(); err == nil {
		t.Fatal("expected error for unknown state")
	}
}

func TestMcpResourceServerState_Valid(t *testing.T) {
	if !McpResourceServerActive.Valid() {
		t.Error("Active should be valid")
	}
	if !McpResourceServerDisabled.Valid() {
		t.Error("Disabled should be valid")
	}
	if McpResourceServerState("Bogus").Valid() {
		t.Error("Bogus should not be valid")
	}
}

func TestMcpResourceServer_IsActive(t *testing.T) {
	m := validMcpResourceServer()
	if !m.IsActive() {
		t.Error("expected Active state to report IsActive() == true")
	}
	m.State = McpResourceServerDisabled
	if m.IsActive() {
		t.Error("expected Disabled state to report IsActive() == false")
	}
}

//spec:covers REQ-OAUTH2-064: 規則の形、重複、スコープの範囲、件数の上限
func TestMcpResourceServerValidateAccessRules(t *testing.T) {
	valid := ResourceAccessRule{ResourceType: "document", Relation: "viewer", Scopes: []string{"mcp.read"}}
	many := make([]ResourceAccessRule, MaxResourceAccessRules+1)
	for i := range many {
		many[i] = ResourceAccessRule{ResourceType: "document", Relation: "r" + strconv.Itoa(i), Scopes: []string{"mcp.read"}}
	}
	cases := []struct {
		name  string
		rules []ResourceAccessRule
		ok    bool
	}{
		{"正常", []ResourceAccessRule{valid}, true},
		{"64 文字の名前", []ResourceAccessRule{{ResourceType: strings.Repeat("a", 64), Relation: "viewer", Scopes: []string{"mcp.read"}}}, true},
		{"65 文字の名前", []ResourceAccessRule{{ResourceType: strings.Repeat("a", 65), Relation: "viewer", Scopes: []string{"mcp.read"}}}, false},
		{"大文字の型", []ResourceAccessRule{{ResourceType: "Document", Relation: "viewer", Scopes: []string{"mcp.read"}}}, false},
		{"空の関係", []ResourceAccessRule{{ResourceType: "document", Scopes: []string{"mcp.read"}}}, false},
		{"組の重複", []ResourceAccessRule{valid, valid}, false},
		{"空のスコープ", []ResourceAccessRule{{ResourceType: "document", Relation: "viewer"}}, false},
		{"登録にないスコープ", []ResourceAccessRule{{ResourceType: "document", Relation: "viewer", Scopes: []string{"other"}}}, false},
		{"上限の超過", many, false},
		{"上限ちょうど", many[:MaxResourceAccessRules], true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validMcpResourceServer()
			m.AccessRules = tc.rules
			if err := m.Validate(); (err == nil) != tc.ok {
				t.Fatalf("err=%v, want ok=%v", err, tc.ok)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-062: 詳細の省略を全許可と読まないこと
func TestResourceAccessRulePermitsDetails(t *testing.T) {
	rule := ResourceAccessRule{Relation: "viewer", AuthorizationDetailType: "doc"}
	matching := spec.AuthorizationDetail{Type: "doc", Locations: []string{"https://rs/"}, Actions: []string{"viewer"}}
	identified := matching
	identified.Identifier = "d1"
	withoutLocations := matching
	withoutLocations.Locations = nil
	withoutActions := matching
	withoutActions.Actions = nil
	otherType := matching
	otherType.Type = "other"
	cases := []struct {
		name    string
		detail  spec.AuthorizationDetail
		permits bool
	}{
		{"一致", matching, true},
		{"identifier の一致", identified, true},
		{"locations の省略", withoutLocations, false},
		{"actions の省略", withoutActions, false},
		{"別の種類", otherType, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rule.PermitsDetails([]spec.AuthorizationDetail{tc.detail}, "https://rs/", "d1"); got != tc.permits {
				t.Fatalf("got %v", got)
			}
		})
	}
	if !(ResourceAccessRule{}).PermitsDetails(nil, "https://rs/", "d1") {
		t.Fatal("種類を指定しない規則は詳細を求めない")
	}
}
