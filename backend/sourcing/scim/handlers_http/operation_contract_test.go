package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func assertScimUnauthorized(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var body map[string]any
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Content-Type") != "application/scim+json" || rec.Header().Get("WWW-Authenticate") == "" || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
		t.Fatalf("authentication refusal status=%d headers=%v body=%s", rec.Code, rec.Header(), rec.Body.String())
	}
	if !reflect.DeepEqual(body["schemas"], []any{scimErrorSchemaURN}) || body["status"] != "401" {
		t.Fatalf("authentication refusal=%v", body)
	}
	if detail, _ := body["detail"].(string); detail == "" {
		t.Fatalf("authentication refusal missing detail: %v", body)
	}
}

//spec:covers RFC7643-SERVICE-PROVIDER-CONFIG: Discovery は Bearer 認証と実装した機能を SCIM の設定として報告する。
//spec:covers REQ-SOURCING-007: Discovery は User、Group と Enterprise 拡張のスキーマと属性を広告する。
//verify:operation GetScimServiceProviderConfig: 200 の SCIM JSON はスキーマ、Bearer 認証、PATCH、filter と未対応の機能を含む。
//verify:operation GetScimResourceTypes: 200 の SCIM JSON は User と Group の ID、名前、エンドポイント、スキーマと Enterprise 拡張を含む。
//verify:operation GetScimSchemas: 200 の SCIM JSON は User、Group と Enterprise 拡張のスキーマと型付きの属性を含む。
func TestScimDiscoveryHTTPContracts(t *testing.T) {
	h := newScimHarness()
	token := issueAllScimToken(t, h.apiTokens)
	rec, config := doScimGet(t, h.echo, token, "/scim/v2/ServiceProviderConfig")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/scim+json" || !reflect.DeepEqual(config["schemas"], []any{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"}) {
		t.Fatalf("config status=%d headers=%v body=%v", rec.Code, rec.Header(), config)
	}
	for _, feature := range []struct {
		name      string
		supported bool
	}{{"patch", true}, {"bulk", false}, {"filter", true}, {"changePassword", false}, {"sort", false}, {"etag", false}} {
		got, ok := config[feature.name].(map[string]any)
		if !ok || got["supported"] != feature.supported {
			t.Fatalf("%s=%v, want supported=%v", feature.name, config[feature.name], feature.supported)
		}
	}
	filter := config["filter"].(map[string]any)
	if filter["maxResults"] != float64(100) {
		t.Fatalf("filter=%v, want maxResults=100", filter)
	}
	schemes, _ := config["authenticationSchemes"].([]any)
	if len(schemes) != 1 {
		t.Fatalf("authenticationSchemes=%v", schemes)
	}
	scheme, _ := schemes[0].(map[string]any)
	if scheme["type"] != "oauthbearertoken" || scheme["name"] == nil || scheme["name"] == "" || scheme["description"] == nil || scheme["description"] == "" {
		t.Fatalf("Bearer authentication scheme=%v", scheme)
	}
	for _, endpoint := range []string{"ResourceTypes", "Schemas"} {
		rec := doScimRequest(t, h.echo, http.MethodGet, "Bearer "+token, "/scim/v2/"+endpoint, nil)
		var items []map[string]any
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/scim+json" || json.Unmarshal(rec.Body.Bytes(), &items) != nil {
			t.Fatalf("%s status=%d body=%s", endpoint, rec.Code, rec.Body.String())
		}
		wantIDs := []string{"urn:ietf:params:scim:schemas:core:2.0:User", enterpriseSchemaURN, "urn:ietf:params:scim:schemas:core:2.0:Group"}
		if endpoint == "ResourceTypes" {
			wantIDs = []string{"User", "Group"}
		}
		if len(items) != len(wantIDs) {
			t.Fatalf("%s=%v, want %d items", endpoint, items, len(wantIDs))
		}
		for _, id := range wantIDs {
			var found map[string]any
			for _, item := range items {
				if item["id"] == id {
					found = item
				}
			}
			if found == nil {
				t.Fatalf("%s missing %s: %v", endpoint, id, items)
			}
			if name, _ := found["name"].(string); name == "" {
				t.Fatalf("%s missing name: %v", endpoint, found)
			}
			if !reflect.DeepEqual(found["schemas"], []any{"urn:ietf:params:scim:schemas:core:2.0:" + strings.TrimSuffix(endpoint, "s")}) {
				t.Fatalf("%s schemas=%v", endpoint, found["schemas"])
			}
			if endpoint == "ResourceTypes" {
				if found["name"] != id || found["endpoint"] != "/"+id+"s" || found["schema"] != "urn:ietf:params:scim:schemas:core:2.0:"+id {
					t.Fatalf("resource type=%v", found)
				}
				if id == "User" && !reflect.DeepEqual(found["schemaExtensions"], []any{map[string]any{"schema": enterpriseSchemaURN, "required": false}}) {
					t.Fatalf("User extensions=%v", found["schemaExtensions"])
				}
			} else {
				attrs, _ := found["attributes"].([]any)
				if len(attrs) == 0 {
					t.Fatalf("schema %s missing attributes", id)
				}
				for _, raw := range attrs {
					attr, _ := raw.(map[string]any)
					for _, field := range []string{"name", "type", "mutability", "returned"} {
						if value, _ := attr[field].(string); value == "" {
							t.Fatalf("attribute missing %s: %v", field, attr)
						}
					}
					if _, ok := attr["uniqueness"].(string); !ok {
						t.Fatalf("attribute missing string uniqueness: %v", attr)
					}
					for _, field := range []string{"multiValued", "required", "caseExact"} {
						if _, ok := attr[field].(bool); !ok {
							t.Fatalf("attribute missing boolean %s: %v", field, attr)
						}
					}
				}
			}
		}
	}
}

//spec:covers RFC7644-BEARER-AUTHORIZATION: Bearer を提示しない SCIM の各操作は認証を拒否する。
//spec:covers RFC7644-ERROR-RESPONSE: 認証拒否は SCIM の schemas、status、detail と WWW-Authenticate を含む。
//verify:operation GetScimServiceProviderConfig: 認証なしの Discovery は 401 の SCIM エラーを返す。
//verify:operation GetScimResourceTypes: 認証なしの型一覧は 401 の SCIM エラーを返す。
//verify:operation GetScimSchemas: 認証なしのスキーマ一覧は 401 の SCIM エラーを返す。
//verify:operation CreateScimUser: 認証なしの User 作成は 401 の SCIM エラーを返す。
//verify:operation ListScimUsers: 認証なしの User 一覧は 401 の SCIM エラーを返す。
//verify:operation GetScimUser: 認証なしの User 取得は 401 の SCIM エラーを返す。
//verify:operation UpdateScimUser: 認証なしの User 置換は 401 の SCIM エラーを返す。
//verify:operation PatchScimUser: 認証なしの User 更新は 401 の SCIM エラーを返す。
//verify:operation DeleteScimUser: 認証なしの User 削除は 401 の SCIM エラーを返す。
//verify:operation CreateScimGroup: 認証なしの Group 作成は 401 の SCIM エラーを返す。
//verify:operation ListScimGroups: 認証なしの Group 一覧は 401 の SCIM エラーを返す。
//verify:operation GetScimGroup: 認証なしの Group 取得は 401 の SCIM エラーを返す。
//verify:operation UpdateScimGroup: 認証なしの Group 置換は 401 の SCIM エラーを返す。
//verify:operation PatchScimGroup: 認証なしの Group 更新は 401 の SCIM エラーを返す。
//verify:operation DeleteScimGroup: 認証なしの Group 削除は 401 の SCIM エラーを返す。
func TestScimOperationsRefuseUnauthenticatedRequests(t *testing.T) {
	h := newScimHarness()
	for _, endpoint := range []string{"ServiceProviderConfig", "ResourceTypes", "Schemas"} {
		assertScimUnauthorized(t, doScimRequest(t, h.echo, http.MethodGet, "", "/scim/v2/"+endpoint, nil))
	}
	for _, kind := range []string{"Users", "Groups"} {
		for _, request := range []struct{ method, suffix string }{{http.MethodGet, ""}, {http.MethodPost, ""}, {http.MethodGet, "/missing"}, {http.MethodPut, "/missing"}, {http.MethodPatch, "/missing"}, {http.MethodDelete, "/missing"}} {
			assertScimUnauthorized(t, doScimRequest(t, h.echo, request.method, "", "/scim/v2/"+kind+request.suffix, map[string]any{}))
		}
	}
}

//spec:covers REQ-SOURCING-004: Group の PATCH は指定属性だけを変更し、再読に反映する。
//verify:operation PatchScimGroup: displayName の PATCH は 200 の同じ ID と変更後の名前を返し、GET の本文に一致する。
func TestScimGroupPatchHTTPContract(t *testing.T) {
	h := newScimHarness()
	token := issueAllScimToken(t, h.apiTokens)
	created, body := doScimJSON(t, h.echo, http.MethodPost, token, "/scim/v2/Groups", map[string]any{"displayName": "before"})
	id, _ := body["id"].(string)
	if created.Code != http.StatusCreated || id == "" {
		t.Fatalf("create status=%d body=%v", created.Code, body)
	}
	path := "/scim/v2/Groups/" + id
	rec, patched := doScimJSON(t, h.echo, http.MethodPatch, token, path, patchOp("replace", "displayName", "after"))
	if rec.Code != http.StatusOK || patched["id"] != id || patched["displayName"] != "after" || rec.Header().Get("Content-Type") != "application/scim+json" {
		t.Fatalf("patch status=%d body=%v", rec.Code, patched)
	}
	read, got := doScimGet(t, h.echo, token, path)
	if read.Code != http.StatusOK || !reflect.DeepEqual(got, patched) {
		t.Fatalf("read status=%d body=%v, want %v", read.Code, got, patched)
	}
}
