package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func oauthContractJSON(t *testing.T, rec *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if rec.Code != status || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body == nil {
		t.Fatalf("JSON=%v err=%v", body, err)
	}
	return body
}

func oauthContractTime(t *testing.T, body map[string]any, key string) {
	t.Helper()
	value, ok := body[key].(string)
	if parsed, err := time.Parse(time.RFC3339Nano, value); !ok || err != nil || parsed.IsZero() {
		t.Fatalf("%s=%v", key, body[key])
	}
}

func oauthContractProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status || rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	body := problemWithoutInstance(t, rec.Body.Bytes())
	if body["type"] != "urn:idmagic:error:"+code || body["status"] != float64(status) {
		t.Fatalf("Problem=%v", body)
	}
}

//verify:operation CreateAdminMcpResourceServer: MCP resource server の作成、取得、更新の HTTP 応答と保存結果
//verify:operation GetAdminMcpResourceServer: MCP resource server の作成、取得、更新の HTTP 応答と保存結果
//verify:operation UpdateAdminMcpResourceServer: MCP resource server の作成、取得、更新の HTTP 応答と保存結果
func TestMcpResourceServerAdminHTTPContracts(t *testing.T) {
	e := newAdminMcpResourceServerHandler()
	csrf, cookie := adminCSRF(t, e)
	created := oauthContractJSON(t, adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/mcp-resource-servers", csrf, cookie, validMcpResourceServerPayload()), http.StatusCreated)
	id, ok := created["id"].(string)
	if !ok || id == "" || created["tenant_id"] != tenancydomain.DefaultTenantID || created["resource"] != "https://mcp.example.com/tools/github" || created["name"] != "GitHub MCP Tools" || created["state"] != "Active" || !reflect.DeepEqual(created["scopes"], []any{"mcp.read", "mcp.write"}) {
		t.Fatalf("created=%v", created)
	}
	oauthContractTime(t, created, "created_at")
	oauthContractTime(t, created, "updated_at")
	get := oauthContractJSON(t, adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/mcp-resource-servers/"+id, csrf, cookie, nil), http.StatusOK)
	if !reflect.DeepEqual(get, created) {
		t.Fatalf("get=%v created=%v", get, created)
	}
	updated := oauthContractJSON(t, adminJSONRequest(t, e, http.MethodPatch, "/api/admin/v1/mcp-resource-servers/"+id, csrf, cookie, map[string]any{"name": "Contract renamed", "scopes": []string{"mcp.read"}, "state": "Disabled"}), http.StatusOK)
	if updated["id"] != id || updated["tenant_id"] != created["tenant_id"] || updated["resource"] != created["resource"] || updated["created_at"] != created["created_at"] || updated["name"] != "Contract renamed" || updated["state"] != "Disabled" || !reflect.DeepEqual(updated["scopes"], []any{"mcp.read"}) {
		t.Fatalf("updated=%v", updated)
	}
	oauthContractTime(t, updated, "updated_at")
	get = oauthContractJSON(t, adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/mcp-resource-servers/"+id, csrf, cookie, nil), http.StatusOK)
	if !reflect.DeepEqual(get, updated) {
		t.Fatalf("update was not saved: get=%v updated=%v", get, updated)
	}
}

//verify:operation CreateAuthorizationDetailType: authorization detail type の作成、取得、更新の HTTP 応答と保存結果
//verify:operation GetAuthorizationDetailType: authorization detail type の作成、取得、更新の HTTP 応答と保存結果
//verify:operation UpdateAuthorizationDetailType: authorization detail type の作成、取得、更新の HTTP 応答と保存結果
func TestAuthorizationDetailTypeAdminHTTPContracts(t *testing.T) {
	e := newAdminAuthzTypeHandler()
	csrf, cookie := adminCSRF(t, e)
	created := oauthContractJSON(t, adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/authorization-detail-types", csrf, cookie, validTypePayload()), http.StatusCreated)
	if created["tenant_id"] != tenancydomain.DefaultTenantID || created["type"] != "data_access" || created["description"] != "文書アクセス" || created["display_template"] != "{datatypes} を {actions}" || created["state"] != "Enabled" {
		t.Fatalf("created=%v", created)
	}
	rules := created["schema"].(map[string]any)["rules"].([]any)
	if len(rules) != 2 || rules[0].(map[string]any)["name"] != "actions" || rules[0].(map[string]any)["semantics"] != "set" || rules[0].(map[string]any)["required"] != true || !reflect.DeepEqual(rules[0].(map[string]any)["allowed"], []any{"read", "write"}) || rules[1].(map[string]any)["name"] != "datatypes" {
		t.Fatalf("schema=%v", created["schema"])
	}
	oauthContractTime(t, created, "created_at")
	oauthContractTime(t, created, "updated_at")
	get := oauthContractJSON(t, adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/authorization-detail-types/data_access", csrf, cookie, nil), http.StatusOK)
	if !reflect.DeepEqual(get, created) {
		t.Fatalf("get=%v created=%v", get, created)
	}
	payload := validTypePayload()
	delete(payload, "type")
	payload["description"] = "Updated contract"
	payload["state"] = "Disabled"
	updated := oauthContractJSON(t, adminJSONRequest(t, e, http.MethodPatch, "/api/admin/v1/authorization-detail-types/data_access", csrf, cookie, payload), http.StatusOK)
	if updated["type"] != created["type"] || updated["tenant_id"] != created["tenant_id"] || updated["created_at"] != created["created_at"] || updated["description"] != "Updated contract" || updated["state"] != "Disabled" || updated["display_template"] != created["display_template"] || !reflect.DeepEqual(updated["schema"], created["schema"]) {
		t.Fatalf("updated=%v", updated)
	}
	oauthContractTime(t, updated, "updated_at")
	get = oauthContractJSON(t, adminJSONRequest(t, e, http.MethodGet, "/api/admin/v1/authorization-detail-types/data_access", csrf, cookie, nil), http.StatusOK)
	if !reflect.DeepEqual(get, updated) {
		t.Fatalf("update was not saved: get=%v updated=%v", get, updated)
	}
}

//verify:operation CreateAdminOAuth2Client: 非管理者への 403 Problem Details
//verify:operation GetAdminOAuth2Client: 非管理者への 403 Problem Details
//verify:operation UpdateAdminOAuth2Client: 非管理者への 403 Problem Details
//verify:operation DeleteAdminOAuth2Client: 非管理者への 403 Problem Details
//verify:operation CreateAuthorizationDetailType: 非管理者への 403 Problem Details
//verify:operation GetAuthorizationDetailType: 非管理者への 403 Problem Details
//verify:operation UpdateAuthorizationDetailType: 非管理者への 403 Problem Details
//verify:operation CreateAdminMcpResourceServer: 非管理者への 403 Problem Details
//verify:operation GetAdminMcpResourceServer: 非管理者への 403 Problem Details
//verify:operation UpdateAdminMcpResourceServer: 非管理者への 403 Problem Details
//verify:operation GetAdminConsent: 非管理者への 403 Problem Details
//verify:operation RevokeAdminConsent: 非管理者への 403 Problem Details
func TestOAuthAdminHTTPContractsRefuseNonAdmin(t *testing.T) {
	clients, _, _ := newAdminOAuth2ClientHandler(t)
	consents, _, _ := newAdminConsentHandler()
	for _, request := range []struct {
		server *echo.Echo
		method string
		path   string
	}{
		{clients, http.MethodPost, "/api/admin/v1/clients"},
		{clients, http.MethodGet, "/api/admin/v1/clients/missing"},
		{clients, http.MethodPatch, "/api/admin/v1/clients/missing"},
		{clients, http.MethodDelete, "/api/admin/v1/clients/missing"},
		{newAdminAuthzTypeHandler(), http.MethodPost, "/api/admin/v1/authorization-detail-types"},
		{newAdminAuthzTypeHandler(), http.MethodGet, "/api/admin/v1/authorization-detail-types/missing"},
		{newAdminAuthzTypeHandler(), http.MethodPatch, "/api/admin/v1/authorization-detail-types/missing"},
		{newAdminMcpResourceServerHandler(), http.MethodPost, "/api/admin/v1/mcp-resource-servers"},
		{newAdminMcpResourceServerHandler(), http.MethodGet, "/api/admin/v1/mcp-resource-servers/missing"},
		{newAdminMcpResourceServerHandler(), http.MethodPatch, "/api/admin/v1/mcp-resource-servers/missing"},
		{consents, http.MethodGet, "/api/admin/v1/consents/alice/portal"},
		{consents, http.MethodDelete, "/api/admin/v1/consents/alice/portal"},
	} {
		csrf, cookie := adminCSRF(t, request.server)
		req := httptest.NewRequest(request.method, "/realms/default"+request.path, http.NoBody)
		req.Header.Set("X-Demo-Sub", "regular")
		req.Header.Set("Origin", "http://idp.test")
		req.Header.Set("X-Csrf-Token", csrf)
		req.AddCookie(cookie)
		rec := httptest.NewRecorder()
		request.server.ServeHTTP(rec, req)
		oauthContractProblem(t, rec, http.StatusForbidden, "access_denied")
	}
}

//verify:operation GetAdminOAuth2Client: 存在しない管理対象への 404 Problem Details
//verify:operation GetAuthorizationDetailType: 存在しない管理対象への 404 Problem Details
//verify:operation GetAdminMcpResourceServer: 存在しない管理対象への 404 Problem Details
func TestOAuthAdminHTTPContractsRefuseMissingResources(t *testing.T) {
	clients, _, _ := newAdminOAuth2ClientHandler(t)
	for _, resource := range []struct {
		server     *echo.Echo
		path, code string
	}{
		{clients, "/api/admin/v1/clients/missing", "client_not_found"},
		{newAdminAuthzTypeHandler(), "/api/admin/v1/authorization-detail-types/missing", "type_not_found"},
		{newAdminMcpResourceServerHandler(), "/api/admin/v1/mcp-resource-servers/missing", "resource_server_not_found"},
	} {
		csrf, cookie := adminCSRF(t, resource.server)
		oauthContractProblem(t, adminJSONRequest(t, resource.server, http.MethodGet, resource.path, csrf, cookie, nil), http.StatusNotFound, resource.code)
	}
}
