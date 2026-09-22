package handlers_http_test

import (
	"net/http"
	"reflect"
	"testing"
)

// scimPatchTarget は PATCH の効果を読む資源を 1 つ作り、その SCIM 上のパスを返す。
type scimPatchTarget struct {
	name   string
	create func(t *testing.T, h scimHarness, tokenStr string) string
}

func scimPatchTargets() []scimPatchTarget {
	return []scimPatchTarget{
		{name: "User", create: func(t *testing.T, h scimHarness, tokenStr string) string {
			t.Helper()
			return "/scim/v2/Users/" + createScimUserForTest(t, h.echo, tokenStr, map[string]any{
				"userName": "target@example.com",
				"name":     map[string]any{"givenName": "Tara", "familyName": "Get"},
				"emails":   []any{map[string]any{"value": "target@example.com", "primary": true}},
				"active":   true,
			})
		}},
		{name: "Group", create: func(t *testing.T, h scimHarness, tokenStr string) string {
			t.Helper()
			member := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "member@example.com"})
			return "/scim/v2/Groups/" + createScimGroupForTest(t, h, tokenStr, map[string]any{
				"displayName": "Target",
				"members":     []map[string]any{{"value": member}},
			})
		}},
	}
}

// 各パスを置換した後に資源を読み直し、置換した属性だけが変わったことを読む。meta は
// 書き込みのたびに lastModified が進むので比較から外す。
//
//spec:covers EX-SOURCING-004-01: User の userName・name・active・emails と Group の displayName・members への replace が、読み直した資源のうち対象属性だけを変え、他の属性を変えない。
func TestScimPatchReplace_ChangesOnlyTheTargetAttribute(t *testing.T) {
	targets := scimPatchTargets()
	fixed := func(value any) func(*testing.T, scimHarness, string) any {
		return func(*testing.T, scimHarness, string) any { return value }
	}
	cases := []struct {
		target int
		path   string
		// value は置換値を返す。members の置換値は同じハーネスにいる User を指す必要がある。
		value func(t *testing.T, h scimHarness, tokenStr string) any
	}{
		{target: 0, path: "userName", value: fixed("renamed@example.com")},
		{target: 0, path: "name", value: fixed(map[string]any{"givenName": "Nora", "familyName": "Named"})},
		{target: 0, path: "active", value: fixed(false)},
		{target: 0, path: "emails", value: fixed([]any{map[string]any{"value": "replaced@example.com", "primary": true}})},
		{target: 1, path: "displayName", value: fixed("Renamed")},
		{target: 1, path: "members", value: func(t *testing.T, h scimHarness, tokenStr string) any {
			t.Helper()
			replacement := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "replacement@example.com"})
			return []map[string]any{{"value": replacement}}
		}},
	}
	for _, tc := range cases {
		target := targets[tc.target]
		t.Run(target.name+" "+tc.path, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			resourcePath := target.create(t, h, tokenStr)
			value := tc.value(t, h, tokenStr)
			_, before := doScimGet(t, h.echo, tokenStr, resourcePath)

			rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, resourcePath, patchOp("replace", tc.path, value))
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
			}
			_, after := doScimGet(t, h.echo, tokenStr, resourcePath)
			if reflect.DeepEqual(after[tc.path], before[tc.path]) {
				t.Errorf("%s = %v, want the replacement to be persisted", tc.path, after[tc.path])
			}
			for attr, value := range before {
				if attr == tc.path || attr == "meta" {
					continue
				}
				if !reflect.DeepEqual(after[attr], value) {
					t.Errorf("%s changed from %v to %v by a replace of %s", attr, value, after[attr], tc.path)
				}
			}
			for attr := range after {
				if _, existed := before[attr]; !existed {
					t.Errorf("%s = %v appeared by a replace of %s", attr, after[attr], tc.path)
				}
			}
		})
	}
}

// 拒否の応答と、拒否の後に読み直した資源が拒否の前と同じであることを対で読む。
//
//spec:covers EX-SOURCING-004-02: User と Group の許可リスト外のパスと存在しない属性のパスへの PATCH を 400 invalidPath で拒否し、読み直した資源が拒否の前と同じである。
//spec:covers EX-SOURCING-004-03: User と Group の id・meta・schemas への PATCH を 400 mutability で拒否し、読み直した資源が拒否の前と同じである。
//spec:covers EX-SOURCING-004-04: User と Group への add・replace・remove 以外の op を 400 invalidValue で拒否し、読み直した資源が拒否の前と同じである。
func TestScimPatch_RefusesUnsupportedPathsAndOpsAndLeavesTheResourceUnchanged(t *testing.T) {
	cases := []struct {
		name         string
		op           string
		path         map[string]string
		value        any
		wantScimType string
	}{
		{name: "path outside the allowlist", op: "replace", path: map[string]string{"User": "nickName", "Group": "description"}, value: "x", wantScimType: "invalidPath"},
		{name: "path to no attribute", op: "replace", path: map[string]string{"User": "noSuchAttribute", "Group": "noSuchAttribute"}, value: "x", wantScimType: "invalidPath"},
		{name: "readOnly id", op: "replace", path: map[string]string{"User": "id", "Group": "id"}, value: "taken-over", wantScimType: "mutability"},
		{name: "readOnly meta", op: "replace", path: map[string]string{"User": "meta", "Group": "meta"}, value: map[string]any{"resourceType": "Other"}, wantScimType: "mutability"},
		{name: "readOnly schemas", op: "replace", path: map[string]string{"User": "schemas", "Group": "schemas"}, value: []any{}, wantScimType: "mutability"},
		{name: "op delete", op: "delete", path: map[string]string{"User": "userName", "Group": "displayName"}, value: "x", wantScimType: "invalidValue"},
		{name: "op move", op: "move", path: map[string]string{"User": "userName", "Group": "displayName"}, value: "x", wantScimType: "invalidValue"},
	}
	for _, target := range scimPatchTargets() {
		for _, tc := range cases {
			t.Run(target.name+" "+tc.name, func(t *testing.T) {
				h := newScimHarness()
				tokenStr := issueAllScimToken(t, h.apiTokens)
				resourcePath := target.create(t, h, tokenStr)
				_, before := doScimGet(t, h.echo, tokenStr, resourcePath)

				rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, resourcePath, patchOp(tc.op, tc.path[target.name], tc.value))
				if rec.Code != http.StatusBadRequest || body["scimType"] != tc.wantScimType {
					t.Fatalf("expected 400 %s, got %d body=%v", tc.wantScimType, rec.Code, body)
				}
				assertScimResourceUnchanged(t, h, tokenStr, resourcePath, before)
			})
		}
	}
}

// createScimGroupForTest は Group を SCIM の POST で作り、その SCIM id を返す。
func createScimGroupForTest(t *testing.T, h scimHarness, tokenStr string, body map[string]any) string {
	t.Helper()
	rec, created := doScimJSON(t, h.echo, http.MethodPost, tokenStr, "/scim/v2/Groups", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: create group: expected 201, got %d body=%v", rec.Code, created)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("setup: create group: expected a server-assigned id, got %v", created)
	}
	return id
}
