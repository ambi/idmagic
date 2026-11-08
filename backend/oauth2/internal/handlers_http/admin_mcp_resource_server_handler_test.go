package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"

	"github.com/ambi/idmagic/backend/idmanagement"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"

	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func newAdminMcpResourceServerHandler() *echo.Echo {
	return newAdminMcpResourceServerHandlerWith(oauth2.NewMemoryModule().ClientRepo)
}

// reports はテストがカタログを置いて読み直すための共有の保存先である。
var catalogs = oauth2memory.NewMcpToolCatalogRepository()

func newAdminMcpResourceServerHandlerWith(clients oauthports.OAuth2ClientRepository, supplied ...oauthports.McpResourceServerRepository) *echo.Echo {
	users := idmanagement.NewMemoryModule().UserRepo
	var servers oauthports.McpResourceServerRepository = oauth2memory.NewMcpResourceServerRepository()
	if len(supplied) > 0 {
		servers = supplied[0]
	}
	now := time.Now().UTC()
	_ = users.Save(context.Background(), &userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	_ = users.Save(context.Background(), &userdomain.User{
		ID: "regular", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "regular",
		PasswordHash: "unused", CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test",
		Emit:   func(spec.DomainEvent) {}, UserRepo: users, OAuth2: oauth2.Module{McpResourceServerRepo: servers, McpToolCatalogRepo: catalogs, ClientRepo: clients},
		AuthnResolver: authentication.DemoHeaderResolver{},
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

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	if createdBody["id"] == "" || createdBody["id"] == nil {
		t.Fatalf("expected id to be generated: %+v", createdBody)
	}
	if createdBody["state"] != "Active" {
		t.Fatalf("expected default state Active, got %v", createdBody["state"])
	}

	listReq := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/mcp-resource-servers", http.NoBody)
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

	first := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status=%d body=%s", first.Code, first.Body.String())
	}
	second := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	if second.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate resource, got %d body=%s", second.Code, second.Body.String())
	}
	listReq := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/mcp-resource-servers", http.NoBody)
	listReq.Header.Set("X-Demo-Sub", "admin")
	listRes := httptest.NewRecorder()
	e.ServeHTTP(listRes, listReq)
	var list struct {
		ResourceServers []map[string]any `json:"resource_servers"`
	}
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRes.Code, listRes.Body.String())
	}
	if err := json.Unmarshal(listRes.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.ResourceServers) != 1 {
		t.Fatalf("duplicate refusal saved another resource server: %+v", list.ResourceServers)
	}
}

func TestAdminUpdatesMcpResourceServerNameScopesState(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	id := createdBody["id"].(string)

	update := adminJSONRequest(t, e, http.MethodPatch, "/api/admin/v1/mcp-resource-servers/"+id, csrf, cookie, map[string]any{
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

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	id := createdBody["id"].(string)

	deleteRes := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/v1/mcp-resource-servers/"+id, csrf, cookie, nil)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleteRes.Code, deleteRes.Body.String())
	}

	assertMcpResourceServerAbsent(t, e, id)
}

func assertMcpResourceServerAbsent(t *testing.T, e *echo.Echo, id string) {
	t.Helper()
	getReq := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/mcp-resource-servers/"+id, http.NoBody)
	getReq.Header.Set("X-Demo-Sub", "admin")
	getRes := httptest.NewRecorder()
	e.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRes.Code)
	}
}

func TestAdminMcpResourceServerRequiresAdmin(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	req := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/mcp-resource-servers", http.NoBody)
	req.Header.Set("X-Demo-Sub", "regular")
	res := httptest.NewRecorder()
	e.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("non-admin must not list resource servers, got %d", res.Code)
	}
}

//spec:covers REQ-OAUTH2-064: 束縛と規則の往復、省略した更新、不正な束縛と規則の拒否
//verify:operation CreateAdminMcpResourceServer: client_id と access_rules の本文、拒否の状態コード
//verify:operation UpdateAdminMcpResourceServer: client_id と access_rules の本文、拒否の状態コード
func TestAdminMcpResourceServerAccessRules(t *testing.T) {
	clients := oauth2.NewMemoryModule().ClientRepo
	for _, clientID := range []string{"rs-a", "rs-b"} {
		if err := clients.Save(context.Background(), &oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: clientID, ClientType: spec.ClientConfidential,
			TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
			GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
		}); err != nil {
			t.Fatal(err)
		}
	}
	e := newAdminMcpResourceServerHandlerWith(clients)
	csrf, cookie := adminCSRF(t, e)
	rules := []map[string]any{{"resource_type": "document", "relation": "viewer", "scopes": []string{"mcp.read"}}}
	payload := validMcpResourceServerPayload()
	payload["client_id"] = "rs-a"
	payload["access_rules"] = rules

	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, payload)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var body struct {
		ID          string           `json:"id"`
		ClientID    string           `json:"client_id"`
		AccessRules []map[string]any `json:"access_rules"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ClientID != "rs-a" || len(body.AccessRules) != 1 {
		t.Fatalf("created=%s", created.Body.String())
	}

	path := "/api/admin/v1/mcp-resource-servers/" + body.ID
	updated := adminJSONRequest(t, e, http.MethodPatch, path, csrf, cookie, map[string]any{"name": "renamed", "scopes": []string{"mcp.read"}})
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"client_id":"rs-a"`) || !strings.Contains(updated.Body.String(), `"relation":"viewer"`) {
		t.Fatalf("省略した更新: status=%d body=%s", updated.Code, updated.Body.String())
	}

	other := validMcpResourceServerPayload()
	other["resource"] = "https://mcp.example.com/tools/other"
	other["client_id"] = "rs-a"
	if rec := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, other); rec.Code != http.StatusBadRequest {
		t.Fatalf("束縛の重複: status=%d", rec.Code)
	}
	other["client_id"] = "unknown"
	if rec := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, other); rec.Code != http.StatusBadRequest {
		t.Fatalf("未知のクライアント: status=%d", rec.Code)
	}
	other["client_id"] = "rs-b"
	other["access_rules"] = []map[string]any{{"resource_type": "document", "relation": "viewer", "scopes": []string{"not-allowed"}}}
	if rec := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, other); rec.Code != http.StatusBadRequest {
		t.Fatalf("範囲外のスコープ: status=%d", rec.Code)
	}
}

func getCatalogTools(t *testing.T, e *echo.Echo, id string) (int, map[string]any) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/mcp-resource-servers/"+id+"/tool-catalog", http.NoBody)
	request.Header.Set("X-Demo-Sub", "admin")
	recorder := httptest.NewRecorder()
	e.ServeHTTP(recorder, request)
	body := map[string]any{}
	_ = json.Unmarshal(recorder.Body.Bytes(), &body)
	return recorder.Code, body
}

//spec:covers REQ-OAUTH2-087: カタログは登録とは別に返り、未受信は空、登録の削除でカタログも消え、存在しない登録は 404 になる。
//verify:operation GetAdminMcpResourceServerToolCatalog: 200 JSON の tools、complete、received_at と、存在しない登録の 404 を観測する。
func TestAdminReadsToolCatalogWithoutPublishingIt(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)
	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	id := createdBody["id"].(string)

	status, body := getCatalogTools(t, e, id)
	if status != http.StatusOK || len(body["tools"].([]any)) != 0 || body["received_at"] != nil {
		t.Fatalf("before a catalog: status=%d body=%v", status, body)
	}
	receivedAt := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	if err := catalogs.Save(context.Background(), &oauthdomain.McpToolCatalog{
		TenantID: tenancydomain.DefaultTenantID, ResourceServerID: id, Complete: true, ReceivedAt: receivedAt,
		Tools: []oauthdomain.CatalogTool{{Name: "read_document", Title: "閲覧", Arguments: []string{"document_id"}}},
	}); err != nil {
		t.Fatal(err)
	}
	status, body = getCatalogTools(t, e, id)
	tools := body["tools"].([]any)
	if status != http.StatusOK || len(tools) != 1 || tools[0].(map[string]any)["name"] != "read_document" || body["complete"] != true || body["received_at"] != receivedAt.Format(time.RFC3339) {
		t.Fatalf("after a catalog: status=%d body=%v", status, body)
	}
	if published := createdBody["access_rules"].([]any); len(published) != 0 {
		t.Fatalf("the catalog published rules: %v", published)
	}

	if deleted := adminJSONRequest(t, e, http.MethodDelete, "/api/admin/v1/mcp-resource-servers/"+id, csrf, cookie, nil); deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d", deleted.Code)
	}
	if catalog, err := catalogs.Find(context.Background(), tenancydomain.DefaultTenantID, id); err != nil || catalog != nil {
		t.Fatalf("catalog survived the registration: %+v %v", catalog, err)
	}
	if status, _ := getCatalogTools(t, e, id); status != http.StatusNotFound {
		t.Fatalf("missing registration status=%d", status)
	}
}

//spec:covers REQ-OAUTH2-087: 内容の変わったカタログの後は changes に変更の時刻、追加、削除、定義変更と直前の定義を返し、直前の一覧がなければ changes を含めない。
//verify:operation GetAdminMcpResourceServerToolCatalog: 200 JSON の changes の changed_at、added、removed、modified を観測する。
func TestAdminReadsReportChangesSinceThePreviousList(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)
	created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload())
	var createdBody map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &createdBody); err != nil {
		t.Fatal(err)
	}
	id := createdBody["id"].(string)
	save := func(at time.Time, tools ...oauthdomain.CatalogTool) {
		t.Helper()
		if err := catalogs.Save(context.Background(), &oauthdomain.McpToolCatalog{
			TenantID: tenancydomain.DefaultTenantID, ResourceServerID: id, Complete: true, ReceivedAt: at, Tools: tools,
		}); err != nil {
			t.Fatal(err)
		}
	}
	first := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	save(first, oauthdomain.CatalogTool{Name: "read_document", Description: "文書を読む", Arguments: []string{"document_id"}}, oauthdomain.CatalogTool{Name: "delete_document"})
	if _, body := getCatalogTools(t, e, id); body["changes"] != nil {
		t.Fatalf("the first report has changes: %v", body["changes"])
	}

	changed := first.Add(time.Hour)
	save(changed, oauthdomain.CatalogTool{Name: "read_document", Description: "文書を社外へ送る", Arguments: []string{"document_id"}}, oauthdomain.CatalogTool{Name: "write_document"})
	status, body := getCatalogTools(t, e, id)
	want := map[string]any{
		"changed_at": changed.Format(time.RFC3339),
		"added":      []any{"write_document"},
		"removed":    []any{"delete_document"},
		"modified": []any{map[string]any{
			"name": "read_document", "fields": []any{"description"},
			"previous": map[string]any{"name": "read_document", "description": "文書を読む", "arguments": []any{"document_id"}},
		}},
	}
	if status != http.StatusOK || !reflect.DeepEqual(body["changes"], want) {
		t.Fatalf("status=%d changes=%#v, want %#v", status, body["changes"], want)
	}
}

//spec:covers REQ-OAUTH2-064: スコープを省いた登録は mcp だけを持ち、明示したスコープはそのまま保つ。
func TestAdminCreatesMcpResourceServerWithTheDefaultScope(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)
	for _, tc := range []struct {
		name   string
		scopes any
		want   []any
	}{{"omitted", nil, []any{"mcp"}}, {"empty", []string{}, []any{"mcp"}}, {"explicit", []string{"tools:read"}, []any{"tools:read"}}} {
		payload := validMcpResourceServerPayload()
		payload["resource"] = "https://default-scope-" + tc.name + ".example/mcp"
		delete(payload, "scopes")
		if tc.scopes != nil {
			payload["scopes"] = tc.scopes
		}
		created := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, payload)
		var body map[string]any
		if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil || created.Code != http.StatusCreated {
			t.Fatalf("%s: status=%d body=%s", tc.name, created.Code, created.Body.String())
		}
		if got := body["scopes"].([]any); len(got) != len(tc.want) || got[0] != tc.want[0] {
			t.Fatalf("%s: scopes=%v, want %v", tc.name, got, tc.want)
		}
	}
}
