package usecases

import (
	"context"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/kernel"
)

func authorizeDepsWithResourceServer(servers ...*domain.McpResourceServer) AuthorizeDeps {
	deps := newAuthorizeDeps(false)
	deps.McpResourceServerRepo = newFakeMcpResourceServerRepo(servers...)
	return deps
}

func TestAuthorize_noResource_unaffected(t *testing.T) {
	in := validAuthorizeInput()
	out, err := Authorize(context.Background(), authorizeDepsWithResourceServer(), in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Request.Resource != nil {
		t.Fatalf("expected nil Resource when not requested, got %v", *out.Request.Resource)
	}
}

func TestAuthorize_registeredActiveResource_boundOnRequest(t *testing.T) {
	rs := &domain.McpResourceServer{
		TenantID: kernel.DefaultTenantID, ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"openid"}, State: domain.McpResourceServerActive,
	}
	in := validAuthorizeInput()
	in.Resource = []string{"https://mcp.example.com/tools"}
	out, err := Authorize(context.Background(), authorizeDepsWithResourceServer(rs), in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Request.Resource == nil || *out.Request.Resource != "https://mcp.example.com/tools" {
		t.Fatalf("expected Resource to be bound, got %v", out.Request.Resource)
	}
}

func TestAuthorize_unregisteredResource_rejectedFailClosed(t *testing.T) {
	in := validAuthorizeInput()
	in.Resource = []string{"https://mcp.example.com/unknown"}
	_, err := Authorize(context.Background(), authorizeDepsWithResourceServer(), in)
	assertOAuthError(t, err, "invalid_target")
}

func TestAuthorize_multipleResources_rejectedFailClosed(t *testing.T) {
	in := validAuthorizeInput()
	in.Resource = []string{"https://mcp.example.com/a", "https://mcp.example.com/b"}
	_, err := Authorize(context.Background(), authorizeDepsWithResourceServer(), in)
	assertOAuthError(t, err, "invalid_target")
}
