package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

type fakeMcpResourceServerRepo struct {
	byResource map[string]*domain.McpResourceServer
}

func newFakeMcpResourceServerRepo(servers ...*domain.McpResourceServer) *fakeMcpResourceServerRepo {
	r := &fakeMcpResourceServerRepo{byResource: map[string]*domain.McpResourceServer{}}
	for _, s := range servers {
		r.byResource[s.TenantID+"|"+s.Resource] = s
	}
	return r
}

func (r *fakeMcpResourceServerRepo) ListByTenant(context.Context, string) ([]*domain.McpResourceServer, error) {
	return nil, nil
}

func (r *fakeMcpResourceServerRepo) FindByID(context.Context, string, string) (*domain.McpResourceServer, error) {
	return nil, nil //nolint:nilnil // unused by ResolveResourceIndicator tests
}

func (r *fakeMcpResourceServerRepo) FindByResource(_ context.Context, tenantID, resource string) (*domain.McpResourceServer, error) {
	return r.byResource[tenantID+"|"+resource], nil
}

func (r *fakeMcpResourceServerRepo) Save(context.Context, *domain.McpResourceServer) error {
	return nil
}

func (r *fakeMcpResourceServerRepo) Delete(context.Context, string, string) error { return nil }

var _ ports.McpResourceServerRepository = (*fakeMcpResourceServerRepo)(nil)

func TestResolveResourceIndicator_noResourceRequested_returnsNilNil(t *testing.T) {
	repo := newFakeMcpResourceServerRepo()
	got, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1", nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil McpResourceServer when no resource requested, got %+v", got)
	}
}

func TestResolveResourceIndicator_multipleResources_rejectedAsInvalidTarget(t *testing.T) {
	repo := newFakeMcpResourceServerRepo()
	_, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1",
		[]string{"https://mcp.example.com/a", "https://mcp.example.com/b"}, nil)
	assertOAuthError(t, err, "invalid_target")
}

func TestResolveResourceIndicator_unregisteredResource_rejectedAsInvalidTarget(t *testing.T) {
	repo := newFakeMcpResourceServerRepo()
	_, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1",
		[]string{"https://mcp.example.com/unknown"}, nil)
	assertOAuthError(t, err, "invalid_target")
}

func TestResolveResourceIndicator_disabledResource_rejectedAsInvalidTarget(t *testing.T) {
	repo := newFakeMcpResourceServerRepo(&domain.McpResourceServer{
		TenantID: "tenant-1", ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read"}, State: domain.McpResourceServerDisabled,
	})
	_, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1",
		[]string{"https://mcp.example.com/tools"}, nil)
	assertOAuthError(t, err, "invalid_target")
}

func TestResolveResourceIndicator_registeredActive_returnsResourceServer(t *testing.T) {
	rs := &domain.McpResourceServer{
		TenantID: "tenant-1", ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read", "mcp.write"}, State: domain.McpResourceServerActive,
	}
	repo := newFakeMcpResourceServerRepo(rs)
	got, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1",
		[]string{"https://mcp.example.com/tools"}, []string{"mcp.read"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got == nil || got.ResourceServerID != "rs-1" {
		t.Fatalf("expected rs-1 to be resolved, got %+v", got)
	}
}

func TestResolveResourceIndicator_scopeExceedsResourceAllowlist_rejectedAsInvalidScope(t *testing.T) {
	rs := &domain.McpResourceServer{
		TenantID: "tenant-1", ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read"}, State: domain.McpResourceServerActive,
	}
	repo := newFakeMcpResourceServerRepo(rs)
	_, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1",
		[]string{"https://mcp.example.com/tools"}, []string{"mcp.read", "mcp.admin"})
	assertOAuthError(t, err, "invalid_scope")
}

func TestResolveResourceIndicator_crossTenantResource_rejectedAsInvalidTarget(t *testing.T) {
	rs := &domain.McpResourceServer{
		TenantID: "tenant-other", ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools", Name: "Tools",
		Scopes: []string{"mcp.read"}, State: domain.McpResourceServerActive,
	}
	repo := newFakeMcpResourceServerRepo(rs)
	_, err := ResolveResourceIndicator(context.Background(), repo, "tenant-1",
		[]string{"https://mcp.example.com/tools"}, nil)
	assertOAuthError(t, err, "invalid_target")
}

func assertOAuthError(t *testing.T, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an OAuthError with code %q, got nil", wantCode)
	}
	oe, ok := errors.AsType[*OAuthError](err)
	if !ok {
		t.Fatalf("expected *OAuthError, got %T: %v", err, err)
	}
	if oe.Code != wantCode {
		t.Fatalf("expected code %q, got %q", wantCode, oe.Code)
	}
}
