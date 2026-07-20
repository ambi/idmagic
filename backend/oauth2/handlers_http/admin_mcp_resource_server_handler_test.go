package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func newAdminMcpResourceServerHandler() *echo.Echo {
	users := usermemory.NewUserRepository()
	servers := oauth2memory.NewMcpResourceServerRepository()
	now := time.Now().UTC()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test",
			Emit:   func(spec.DomainEvent) {},
		}, UserRepo: users, OAuth2: oauth2.Module{McpResourceServerRepo: servers},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e
}

func validMcpResourceServerPayload() map[string]any {
	return map[string]any{
		"resource": "https://mcp.example.com/tools/github",
		"name":     "GitHub MCP Tools",
		"scopes":   []string{"mcp.read", "mcp.write"},
	}
}

func TestAdminCreatesAndListsMcpResourceServer(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	if createdBody["resource_server_id"] == "" || createdBody["resource_server_id"] == nil {
		t.Fatalf("expected resource_server_id to be generated: %+v", createdBody)
	}
	if createdBody["state"] != "Active" {
		t.Fatalf("expected default state Active, got %v", createdBody["state"])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/mcp-resource-servers", http.NoBody)
	listReq.Header.Set("X-Demo-Sub", "admin")
	listRes := httptest.NewRecorder()
	e.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status=%d", listRes.Code)
	}
	var list struct {
		ResourceServers []map[string]any `json:"resource_servers"`
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.ResourceServers) != 1 || list.ResourceServers[0]["resource"] != "https://mcp.example.com/tools/github" {
		t.Fatalf("unexpected list: %+v", list.ResourceServers)
	}
}

func TestAdminRejectsDuplicateResource(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)

	first := adminJSONRequest(t, e, http.MethodPost, "/api/admin/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", first.Code, first.Body.String())
	}
	second := adminJSONRequest(t, e, http.MethodPost, "/api/admin/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	if second.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate resource, got %d body=%s", second.Code, second.Body.String())
	}
}

func TestAdminUpdatesMcpResourceServerNameScopesState(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	id := createdBody["resource_server_id"].(string)

	update := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/mcp-resource-servers/"+id, csrf, cookie, map[string]any{
		"name":   "GitHub Tools (renamed)",
		"scopes": []string{"mcp.read"},
		"state":  "Disabled",
	})
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	var updatedBody map[string]any
	if err := json.Unmarshal(update.Body.Bytes(), &updatedBody); err != nil {
		t.Fatal(err)
	}
	if updatedBody["name"] != "GitHub Tools (renamed)" || updatedBody["state"] != "Disabled" {
		t.Fatalf("update did not apply: %+v", updatedBody)
	}
	// resource (canonical URI) は不変。
	if updatedBody["resource"] != "https://mcp.example.com/tools/github" {
		t.Fatalf("resource must be immutable, got %v", updatedBody["resource"])
	}
}

func TestAdminDeletesMcpResourceServer(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	id := createdBody["resource_server_id"].(string)

	deleteRes := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/mcp-resource-servers/"+id, csrf, cookie, nil)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteRes.Code, deleteRes.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/admin/mcp-resource-servers/"+id, http.NoBody)
	getReq.Header.Set("X-Demo-Sub", "admin")
	getRes := httptest.NewRecorder()
	e.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRes.Code)
	}
}

func TestAdminMcpResourceServerRequiresAdmin(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/mcp-resource-servers", http.NoBody)
	req.Header.Set("X-Demo-Sub", "regular")
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("non-admin must not list resource servers, got %d", res.Code)
	}
}
