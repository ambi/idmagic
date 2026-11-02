package server_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

//spec:covers REQ-SOURCING-002, REQ-SOURCING-004, REQ-SOURCING-025: 公開発行した資格情報による名前の作成、置換、部分更新、拒否後の無変更を SCIM と管理 API の両方から観測する。
//verify:operation CreateScimUser: 201 の名前の投影と不正な型の 400 の SCIM エラーを確認する。
//verify:operation UpdateScimUser: 200 の優先順位と省略した名前の初期化、不正な型の拒否を確認する。
//verify:operation PatchScimUser: 200 の同じ更新先と、型の不正な操作列の 400 および無変更を確認する。
//verify:operation GetScimUser: 200 の二つの名前が正規の名前を示すことを確認する。
//verify:operation ListScimUsers: 検索結果の二つの名前が正規の名前を示すことを確認する。
//verify:operation GetScimSchemas: displayName が単一の書き込み可能な文字列として広告されることを確認する。
func TestSCIMUserDisplayNameReachesAdminProfile(t *testing.T) {
	f := newPublicSetupPath(t, stack.WithSCIM())
	credentials := f.request(t, http.MethodPost, "/api/admin/v1/api-tokens", map[string]any{
		"description": "SCIM name regression", "expiry_days": 1,
		"scopes": []string{"scim:users:read", "scim:users:write"},
	}, f.admin, http.StatusCreated)
	bearer := setupString(t, credentials, "token")
	created := f.request(t, http.MethodPost, "/scim/v2/Users", map[string]any{
		"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "scim-name", "active": true, "displayName": "SCIM owner",
	}, nil, http.StatusCreated, bearer)
	path := "/scim/v2/Users/" + setupString(t, created, "id")
	assertSCIMName(t, created, "SCIM owner")
	observe := func(want string) {
		t.Helper()
		got := f.request(t, http.MethodGet, path, nil, nil, http.StatusOK, bearer)
		assertSCIMName(t, got, want)
		users := f.request(t, http.MethodGet, "/api/admin/v1/users?query=scim-name", nil, f.admin, http.StatusOK)
		user := scimOnlyResult(t, users, "users")
		if user["name"] != want {
			t.Fatalf("administrative name=%v, want %q", user["name"], want)
		}
		list := f.request(t, http.MethodGet, "/scim/v2/Users?filter=userName%20eq%20%22scim-name%22", nil, nil, http.StatusOK, bearer)
		assertSCIMName(t, scimOnlyResult(t, list, "Resources"), want)
	}
	observe("SCIM owner")
	for _, replacement := range []struct {
		body map[string]any
		want string
	}{
		{map[string]any{"userName": "scim-name", "displayName": "Replacement"}, "Replacement"},
		{map[string]any{"userName": "scim-name", "displayName": "Ignored", "name": map[string]any{"formatted": "Existing format"}}, "Existing format"},
		{map[string]any{"userName": "scim-name", "displayName": "Ignored", "name": map[string]any{"formatted": ""}}, ""},
		{map[string]any{"userName": "scim-name", "displayName": "Null fallback", "name": map[string]any{"formatted": nil}}, "Null fallback"},
		{map[string]any{"userName": "scim-name"}, ""},
	} {
		response := f.request(t, http.MethodPut, path, replacement.body, nil, http.StatusOK, bearer)
		assertSCIMName(t, response, replacement.want)
		observe(replacement.want)
	}
	patch := func(operations []map[string]any, wantStatus int) map[string]any {
		t.Helper()
		return f.request(t, http.MethodPatch, path, map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"}, "Operations": operations,
		}, nil, wantStatus, bearer)
	}
	for _, operation := range []struct {
		op, path string
		value    any
		want     string
	}{
		{"add", "displayName", "Added", "Added"},
		{"replace", "displayName", "Changed", "Changed"},
		{"replace", "name.formatted", "Formatted", "Formatted"},
		{"replace", "displayName", nil, ""},
		{"add", "name", map[string]any{"formatted": "Object name"}, "Object name"},
		{"remove", "displayName", nil, ""},
	} {
		response := patch([]map[string]any{{"op": operation.op, "path": operation.path, "value": operation.value}}, http.StatusOK)
		assertSCIMName(t, response, operation.want)
		observe(operation.want)
	}
	patch([]map[string]any{{"op": "replace", "path": "displayName", "value": "First"}, {"op": "replace", "path": "name.formatted", "value": "Last"}}, http.StatusOK)
	observe("Last")
	refused := patch([]map[string]any{{"op": "replace", "path": "displayName", "value": "Must not persist"}, {"op": "replace", "path": "name.formatted", "value": 42}}, http.StatusBadRequest)
	assertSCIMInvalidName(t, refused)
	observe("Last")
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		target, username := path, "scim-name"
		if method == http.MethodPost {
			target, username = "/scim/v2/Users", "invalid-scim-name"
		}
		response := f.request(t, method, target, map[string]any{"userName": username, "displayName": 42}, nil, http.StatusBadRequest, bearer)
		assertSCIMInvalidName(t, response)
	}
	observe("Last")
	invalid := f.request(t, http.MethodGet, "/api/admin/v1/users?query=invalid-scim-name", nil, f.admin, http.StatusOK)
	entries, ok := invalid["users"].([]any)
	if !ok || len(entries) != 0 {
		t.Fatalf("refused creation left a user: %v", invalid)
	}
	req := httptest.NewRequest(http.MethodGet, "/realms/default/scim/v2/Schemas", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+bearer)
	rec := httptest.NewRecorder()
	f.s.Echo.ServeHTTP(rec, req)
	var resources []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resources); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("schemas status=%d body=%s error=%v", rec.Code, rec.Body.String(), err)
	}
	found := false
	for _, schema := range resources {
		if schema["id"] != "urn:ietf:params:scim:schemas:core:2.0:User" {
			continue
		}
		attrs, _ := schema["attributes"].([]any)
		for _, rawAttr := range attrs {
			attr, ok := rawAttr.(map[string]any)
			if !ok || attr["name"] != "displayName" {
				continue
			}
			found = true
			if attr["type"] != "string" || attr["mutability"] != "readWrite" || attr["multiValued"] == true {
				t.Fatalf("displayName schema=%v", attr)
			}
		}
	}
	if !found {
		t.Fatal("User schema does not advertise displayName")
	}
}

func scimOnlyResult(t *testing.T, body map[string]any, key string) map[string]any {
	t.Helper()
	entries, ok := body[key].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("%s=%v, want exactly one result", key, body[key])
	}
	entry, ok := entries[0].(map[string]any)
	if !ok {
		t.Fatalf("%s entry is not an object: %v", key, entries[0])
	}
	return entry
}

func assertSCIMName(t *testing.T, body map[string]any, want string) {
	t.Helper()
	name, ok := body["name"].(map[string]any)
	if !ok || name["formatted"] != want {
		t.Fatalf("SCIM name=%v, want formatted=%q", body["name"], want)
	}
	if want == "" {
		if _, exists := body["displayName"]; exists {
			t.Fatalf("unassigned displayName should be omitted: %v", body["displayName"])
		}
	} else if body["displayName"] != want {
		t.Fatalf("SCIM displayName=%v, want %q", body["displayName"], want)
	}
}

func assertSCIMInvalidName(t *testing.T, body map[string]any) {
	t.Helper()
	if body["status"] != "400" || body["scimType"] != "invalidValue" {
		t.Fatalf("SCIM error=%v, want status=400 scimType=invalidValue", body)
	}
}
