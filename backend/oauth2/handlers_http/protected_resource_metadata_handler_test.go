package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func newProtectedResourceMetadataHandler() *echo.Echo {
	servers := oauth2memory.NewMcpResourceServerRepository()
	servers.Seed(&domain.McpResourceServer{
		TenantID: tenancydomain.DefaultTenantID, ResourceServerID: "rs-1",
		Resource: "https://mcp.example.com/tools/github", Name: "GitHub MCP Tools",
		Scopes: []string{"mcp.read", "mcp.write"}, State: domain.McpResourceServerActive,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "https://idp.example",
			Emit:   func(spec.DomainEvent) {},
		},
		OAuth2: oauth2.Module{McpResourceServerRepo: servers},
	})
	return e
}

func TestProtectedResourceMetadata_registeredResource_returnsMetadata(t *testing.T) {
	e := newProtectedResourceMetadataHandler()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource?resource=https://mcp.example.com/tools/github", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Resource != "https://mcp.example.com/tools/github" {
		t.Fatalf("unexpected resource: %q", body.Resource)
	}
	// support.RequestIssuer はテナント別 issuer (/realms/{tenant}) を解決する (discovery と同方針)。
	if len(body.AuthorizationServers) != 1 || body.AuthorizationServers[0] != "https://idp.example/realms/default" {
		t.Fatalf("unexpected authorization_servers: %v", body.AuthorizationServers)
	}
	if len(body.ScopesSupported) != 2 {
		t.Fatalf("unexpected scopes_supported: %v", body.ScopesSupported)
	}
}

func TestProtectedResourceMetadata_missingResourceParam_returnsRealmAPI(t *testing.T) {
	e := newProtectedResourceMetadataHandler()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Resource string   `json:"resource"`
		Scopes   []string `json:"scopes_supported"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Resource != "https://idp.example/realms/default" || !slices.Contains(body.Scopes, "account:read") {
		t.Fatalf("metadata=%+v", body)
	}
}

func TestProtectedResourceMetadata_unregisteredResource_rejected(t *testing.T) {
	e := newProtectedResourceMetadataHandler()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource?resource=https://mcp.example.com/unknown", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unregistered resource, got %d body=%s", rec.Code, rec.Body.String())
	}
}
