package usecases

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

func TestBuildProtectedResourceMetadata_registeredActive(t *testing.T) {
	rs := &domain.McpResourceServer{
		TenantID: "tenant-1", ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read", "mcp.write"}, State: domain.McpResourceServerActive,
	}
	repo := newFakeMcpResourceServerRepo(rs)
	meta, err := BuildProtectedResourceMetadata(context.Background(), repo, "tenant-1", "https://mcp.example.com/tools", "https://idp.example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if meta.Resource != "https://mcp.example.com/tools" {
		t.Fatalf("unexpected resource: %q", meta.Resource)
	}
	if len(meta.AuthorizationServers) != 1 || meta.AuthorizationServers[0] != "https://idp.example.com" {
		t.Fatalf("unexpected authorization_servers: %v", meta.AuthorizationServers)
	}
	if len(meta.ScopesSupported) != 2 {
		t.Fatalf("unexpected scopes_supported: %v", meta.ScopesSupported)
	}
	if len(meta.BearerMethodsSupported) == 0 {
		t.Fatal("expected bearer_methods_supported to be non-empty")
	}
}

func TestBuildProtectedResourceMetadata_unregistered_rejectedAsInvalidTarget(t *testing.T) {
	repo := newFakeMcpResourceServerRepo()
	_, err := BuildProtectedResourceMetadata(context.Background(), repo, "tenant-1", "https://mcp.example.com/unknown", "https://idp.example.com")
	assertOAuthError(t, err, "invalid_target")
}

func TestBuildProtectedResourceMetadata_disabled_rejectedAsInvalidTarget(t *testing.T) {
	rs := &domain.McpResourceServer{
		TenantID: "tenant-1", ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read"}, State: domain.McpResourceServerDisabled,
	}
	repo := newFakeMcpResourceServerRepo(rs)
	_, err := BuildProtectedResourceMetadata(context.Background(), repo, "tenant-1", "https://mcp.example.com/tools", "https://idp.example.com")
	assertOAuthError(t, err, "invalid_target")
}
