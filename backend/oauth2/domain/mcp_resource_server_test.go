package domain

import (
	"testing"
	"time"
)

func validMcpResourceServer() McpResourceServer {
	now := time.Now().UTC()
	return McpResourceServer{
		TenantID:         "tenant-1",
		ResourceServerID: "11111111-1111-1111-1111-111111111111",
		Resource:         "https://mcp.example.com/tools/github",
		Name:             "GitHub MCP Tools",
		Scopes:           []string{"mcp.read", "mcp.write"},
		State:            McpResourceServerActive,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func TestMcpResourceServerValidate_valid(t *testing.T) {
	m := validMcpResourceServer()
	if err := m.Validate(); err != nil {
		t.Fatalf("expected valid McpResourceServer, got error: %v", err)
	}
}

func TestMcpResourceServerValidate_rejectsEmptyResourceServerID(t *testing.T) {
	m := validMcpResourceServer()
	m.ResourceServerID = ""
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
