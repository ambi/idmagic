package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	approvaldomain "github.com/ambi/idmagic/backend/oauth2/internal/approval/domain"
	oauthmemory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/labstack/echo/v5"
)

func impactAdmin(t *testing.T, module oauth2.Module) (*echo.Echo, *[]map[string]any) {
	t.Helper()
	users := idmanagement.NewMemoryModule().UserRepo
	now := time.Now().UTC()
	if err := users.Save(t.Context(), &userdomain.User{ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	events := []map[string]any{}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{Issuer: "http://idp.test", UserRepo: users, OAuth2: module, AuthnResolver: authentication.DemoHeaderResolver{}, Emit: func(event spec.DomainEvent) {
		if event.EventType() != "PermissionImpactChanged" {
			return
		}
		data, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		events = append(events, body)
	}})
	return e, &events
}

func impactBody(t *testing.T, response *httptest.ResponseRecorder, status int) map[string]any {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body)
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}

func impactGet(t *testing.T, e *echo.Echo, path string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/realms/default"+path, http.NoBody)
	req.Header.Set("X-Demo-Sub", "admin")
	response := httptest.NewRecorder()
	e.ServeHTTP(response, req)
	return impactBody(t, response, 200)
}

//spec:covers REQ-OAUTH2-097: 管理 API で明示した false を保存して読み戻し、省略と解除を区別し、不正値の拒否後は値と監査を変えない。
func TestPermissionImpactRegistration(t *testing.T) {
	for _, kind := range []string{"type", "resource"} {
		t.Run(kind, func(t *testing.T) {
			module := oauth2.NewMemoryModule()
			e, events := impactAdmin(t, module)
			csrf, cookie := adminCSRF(t, e)
			path := "/api/admin/v1/authorization-detail-types"
			payload := validTypePayload()
			key := "impact"
			hint := map[string]any{"changes": "reversible", "personal_data": false}
			if kind == "resource" {
				path = "/api/admin/v1/mcp-resource-servers"
				payload = validMcpResourceServerPayload()
				key = "scope_hints"
				payload[key] = map[string]any{"mcp.read": hint}
			} else {
				payload[key] = hint
			}
			created := impactBody(t, adminJSONRequest(t, e, "POST", path, csrf, cookie, payload), 201)
			id, _ := created["type"].(string)
			if kind == "resource" {
				id = created["id"].(string)
			}
			target := path + "/" + id
			if got := impactGet(t, e, target)[key]; !reflect.DeepEqual(got, payload[key]) {
				t.Fatalf("影響度=%v want=%v", got, payload[key])
			}
			if len(*events) != 1 || (*events)[0]["actorUserId"] != "admin" || (*events)[0]["tenantId"] != tenancydomain.DefaultTenantID || !reflect.DeepEqual((*events)[0]["before"], map[string]any{}) {
				t.Fatalf("作成の監査=%v", *events)
			}
			previous := created[key]
			delete(payload, key)
			updated := impactBody(t, adminJSONRequest(t, e, "PATCH", target, csrf, cookie, payload), 200)
			if !reflect.DeepEqual(updated[key], previous) || len(*events) != 1 {
				t.Fatalf("省略で値か監査が変化: %v %v", updated, *events)
			}
			for _, invalid := range []any{map[string]any{"changes": "none"}, map[string]any{"changes": "unknown", "personal_data": true}, map[string]any{"personal_data": false}, map[string]any{"changes": "none", "personal_data": nil}} {
				payload[key] = invalid
				if kind == "resource" {
					payload[key] = map[string]any{"mcp.read": invalid}
				}
				status := 422
				if kind == "resource" {
					status = 400
				}
				impactBody(t, adminJSONRequest(t, e, "PATCH", target, csrf, cookie, payload), status)
				if !reflect.DeepEqual(impactGet(t, e, target)[key], previous) || len(*events) != 1 {
					t.Fatal("拒否後に値か監査が変化")
				}
			}
			payload[key] = nil
			if kind == "resource" {
				payload[key] = map[string]any{"unregistered": hint}
				impactBody(t, adminJSONRequest(t, e, "PATCH", target, csrf, cookie, payload), 400)
				if !reflect.DeepEqual(impactGet(t, e, target)[key], previous) || len(*events) != 1 {
					t.Fatal("未知のスコープの拒否後に値か監査が変化")
				}
			}
			payload[key] = nil
			if kind == "resource" {
				payload[key] = map[string]any{}
			}
			cleared := impactBody(t, adminJSONRequest(t, e, "PATCH", target, csrf, cookie, payload), 200)
			if value, ok := cleared[key]; ok && value != nil {
				t.Fatalf("解除後=%v", value)
			}
			if len(*events) != 2 || reflect.DeepEqual((*events)[1]["before"], (*events)[1]["after"]) {
				t.Fatalf("解除の監査=%v", *events)
			}
			payload[key] = previous
			impactBody(t, adminJSONRequest(t, e, "PATCH", target, csrf, cookie, payload), 200)
			deleted := adminJSONRequest(t, e, "DELETE", target, csrf, cookie, nil)
			if deleted.Code != 204 || len(*events) != 4 || !reflect.DeepEqual((*events)[3]["after"], map[string]any{}) || (*events)[3]["targetId"] != id {
				t.Fatalf("削除の監査=%v response=%s", *events, deleted.Body)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-098: 管理 API で登録したスコープと種類の影響度が本人の同意と付与前の応答へ届き、別テナントの申告が混ざらない。
func TestConsentPermissionImpact(t *testing.T) {
	module := oauth2.NewMemoryModule()
	admin, _ := impactAdmin(t, module)
	csrf, cookie := adminCSRF(t, admin)
	hint := map[string]any{"changes": "irreversible", "personal_data": true}
	payload := validMcpResourceServerPayload()
	payload["scopes"] = []string{"openid"}
	payload["scope_hints"] = map[string]any{"openid": hint}
	impactBody(t, adminJSONRequest(t, admin, "POST", "/api/admin/v1/mcp-resource-servers", csrf, cookie, payload), 201)
	detail := validTypePayload()
	detail["impact"] = hint
	impactBody(t, adminJSONRequest(t, admin, "POST", "/api/admin/v1/authorization-detail-types", csrf, cookie, detail), 201)
	requests := &consentStateStore{AuthorizationRequestStore: oauthmemory.NewAuthorizationRequestStore()}
	e, _ := newAuthorizeTestServer(t, &authdomain.AuthenticationContext{UserID: "alice", AuthTime: time.Now().Unix(), AMR: []string{"pwd"}}, nil, func(d *httpadapter.Deps) {
		d.OAuth2.RequestStore = requests
		d.OAuth2.McpResourceServerRepo = module.McpResourceServerRepo
		d.OAuth2.AuthzDetailTypeRepo = module.AuthzDetailTypeRepo
	})
	auth := runAuthorize(t, e, authorizeQuery(nil))
	req, err := requests.Find(t.Context(), requests.id)
	if err != nil || req == nil {
		t.Fatalf("同意待ち=%v %v", req, err)
	}
	req.AuthorizationDetails = []spec.AuthorizationDetail{{Type: "data_access", Actions: []string{"read"}}}
	if err := requests.Save(t.Context(), req); err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRequest("GET", "/realms/default/api/auth/transaction", http.NoBody)
	for _, cookie := range auth.Result().Cookies() {
		get.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, get)
	body := impactBody(t, response, 200)
	if !reflect.DeepEqual(body["scope_hints"], map[string]any{"openid": hint}) {
		t.Fatalf("同意のヒント=%v", body)
	}
	views := body["authorization_details"].([]any)
	if len(views) != 1 || !reflect.DeepEqual(views[0].(map[string]any)["impact"], hint) {
		t.Fatalf("種類のヒント=%v", views)
	}
	delegation := impactGet(t, admin, "/api/account/v1/agent-delegations")
	if !reflect.DeepEqual(delegation["scope_hints"], map[string]any{"openid": hint}) {
		t.Fatalf("付与前のヒント=%v", delegation)
	}
	foreign := &oauthdomain.McpResourceServer{TenantID: "other", ID: "other", Resource: "https://other.example", State: oauthdomain.McpResourceServerActive, Scopes: []string{"openid"}}
	if err := module.McpResourceServerRepo.Save(t.Context(), foreign); err != nil {
		t.Fatal(err)
	}
	delegation = impactGet(t, admin, "/api/account/v1/agent-delegations")
	if !reflect.DeepEqual(delegation["scope_hints"], map[string]any{"openid": hint}) {
		t.Fatalf("別テナントの申告が混在=%v", delegation)
	}
}

//spec:covers REQ-OAUTH2-098: 本人の承認一覧へスコープと種類と実際の対象ツールの影響度を運び、同名でも再登録した別のツールの申告は使わない。
func TestApprovalPermissionImpact(t *testing.T) {
	ctx := tenantports.WithTenant(t.Context(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm}, "", "")
	module := oauth2.NewMemoryModule()
	e, _ := impactAdmin(t, module)
	csrf, cookie := adminCSRF(t, e)
	hint := map[string]any{"changes": "irreversible", "personal_data": true}
	payload := validMcpResourceServerPayload()
	payload["scope_hints"] = map[string]any{"mcp.read": hint}
	payload["access_rules"] = []any{map[string]any{"resource_type": "idmagic_mcp_operation", "relation": "invoke", "scopes": []string{"mcp.read"}, "tool": map[string]any{"name": "pay", "display_name": "Pay", "impact": hint}}}
	server := impactBody(t, adminJSONRequest(t, e, "POST", "/api/admin/v1/mcp-resource-servers", csrf, cookie, payload), 201)
	detail := validTypePayload()
	detail["impact"] = hint
	impactBody(t, adminJSONRequest(t, e, "POST", "/api/admin/v1/authorization-detail-types", csrf, cookie, detail), 201)
	serverID := server["id"].(string)
	tool := server["access_rules"].([]any)[0].(map[string]any)["tool"].(map[string]any)
	id, err := spec.NewUUIDv4()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	request := &approvaldomain.ApprovalRequest{ID: id, UserID: "admin", ClientID: "client", TenantID: tenancydomain.DefaultTenantID, State: spec.ApprovalPending, Scopes: []string{"mcp.read", "unknown"}, AuthorizationDetails: []spec.AuthorizationDetail{{Type: "data_access"}}, RequestedAt: now, ExpiresAt: now.Add(time.Minute), ResourceServerID: &serverID, ToolName: "pay", ToolID: tool["id"].(string)}
	if err := module.ApprovalRequestStore.Save(ctx, request); err != nil {
		t.Fatal(err)
	}
	list := func() map[string]any {
		t.Helper()
		return impactGet(t, e, "/api/account/v1/approval-requests")["approval_requests"].([]any)[0].(map[string]any)
	}
	body := list()
	if !reflect.DeepEqual(body["tool_impact"], hint) || !reflect.DeepEqual(body["scope_hints"], map[string]any{"mcp.read": hint}) || !reflect.DeepEqual(body["detail_impacts"], map[string]any{"data_access": hint}) {
		t.Fatalf("承認の影響度=%v", body)
	}
	request.ToolID = "deleted-tool"
	if err := module.ApprovalRequestStore.Save(ctx, request); err != nil {
		t.Fatal(err)
	}
	if body = list(); body["tool_impact"] != nil {
		t.Fatalf("別のツールの申告を使用=%v", body)
	}
}
