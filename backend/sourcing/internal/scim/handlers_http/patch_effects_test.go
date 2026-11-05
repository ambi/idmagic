package handlers_http_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
)

const enterpriseUserURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"

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

// pathlessPatchOp は path を持たない一つの操作の PATCH 本文を作る。Entra ID の無効化と同じく op の先頭を大文字で送る。
func pathlessPatchOp(op string, value any) map[string]any {
	return map[string]any{
		"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
		"Operations": []map[string]any{{"op": op, "value": value}},
	}
}

// path のない操作を送った後に資源を読み直し、value に挙げた属性だけが変わったことを読む。
//
//spec:covers RFC7644-PATCH-PATH-OPTIONAL: path を省いた add と replace を受け入れ、value の各メンバーを、その名前を path にした操作として適用する。
//spec:covers REQ-SOURCING-004: path を省いた操作の value の各メンバーを、path 付きの操作と同じ許可の一覧で適用し、ほかの属性を変えない。
func TestScimPatchWithoutPath_AppliesEachValueMemberAsItsPath(t *testing.T) {
	targets := scimPatchTargets()
	cases := []struct {
		name    string
		target  int
		op      string
		value   map[string]any
		changed []string
	}{
		{name: "User deactivation", target: 0, op: "Replace", value: map[string]any{"active": false}, changed: []string{"active"}},
		{name: "User userName and active", target: 0, op: "replace", value: map[string]any{"userName": "renamed@example.com", "active": false}, changed: []string{"userName", "active"}},
		{name: "User add of enterprise attributes", target: 0, op: "Add", value: map[string]any{
			enterpriseUserURN: map[string]any{"department": "Engineering", "employeeNumber": "701984"},
		}, changed: []string{enterpriseUserURN, "schemas"}},
		{name: "Group rename", target: 1, op: "Replace", value: map[string]any{"displayName": "Renamed"}, changed: []string{"displayName"}},
	}
	for _, tc := range cases {
		target := targets[tc.target]
		t.Run(tc.name, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			resourcePath := target.create(t, h, tokenStr)
			_, before := doScimGet(t, h.echo, tokenStr, resourcePath)

			rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, resourcePath, pathlessPatchOp(tc.op, tc.value))
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
			}
			_, after := doScimGet(t, h.echo, tokenStr, resourcePath)
			for attr, want := range tc.value {
				if !reflect.DeepEqual(after[attr], want) {
					t.Errorf("%s = %v, want %v", attr, after[attr], want)
				}
			}
			for attr, value := range before {
				if attr == "meta" || slices.Contains(tc.changed, attr) {
					continue
				}
				if !reflect.DeepEqual(after[attr], value) {
					t.Errorf("%s changed from %v to %v by a pathless %s", attr, value, after[attr], tc.op)
				}
			}
		})
	}
}

// 拒否の応答と、拒否の後に読み直した資源が拒否の前と同じであることを対で読む。
//
//spec:covers RFC7644-PATCH-PATH-OPTIONAL: path を省いた remove を noTarget で、空でないオブジェクトでない value を invalidValue で拒否する。
//spec:covers REQ-SOURCING-004: path を省いた操作の value の未対応の属性を invalidPath、読み取り専用の属性を mutability で拒否し、リソースを変えない。
func TestScimPatchWithoutPath_RefusesAndLeavesTheResourceUnchanged(t *testing.T) {
	cases := []struct {
		name         string
		op           string
		value        map[string]any
		wantScimType string
	}{
		{name: "remove", op: "remove", value: map[string]any{"User": map[string]any{"active": false}, "Group": map[string]any{"displayName": "x"}}, wantScimType: "noTarget"},
		{name: "string value", op: "replace", value: map[string]any{"User": "x", "Group": "x"}, wantScimType: "invalidValue"},
		{name: "empty object", op: "replace", value: map[string]any{"User": map[string]any{}, "Group": map[string]any{}}, wantScimType: "invalidValue"},
		{name: "null value", op: "add", value: map[string]any{"User": nil, "Group": nil}, wantScimType: "invalidValue"},
		{name: "attribute outside the allowlist", op: "replace", value: map[string]any{"User": map[string]any{"nickName": "x"}, "Group": map[string]any{"description": "x"}}, wantScimType: "invalidPath"},
		{name: "readOnly id beside a supported attribute", op: "replace", value: map[string]any{
			"User":  map[string]any{"active": false, "id": "taken-over"},
			"Group": map[string]any{"displayName": "Renamed", "id": "taken-over"},
		}, wantScimType: "mutability"},
		{name: "type-incompatible value", op: "replace", value: map[string]any{"User": map[string]any{"active": "false"}, "Group": map[string]any{"members": "x"}}, wantScimType: "invalidValue"},
		{name: "enterprise extension that is not an object", op: "add", value: map[string]any{"User": map[string]any{enterpriseUserURN: "Engineering"}}, wantScimType: "invalidValue"},
		{name: "enterprise extension on a Group", op: "add", value: map[string]any{"Group": map[string]any{enterpriseUserURN: map[string]any{"department": "Engineering"}}}, wantScimType: "invalidPath"},
	}
	for _, target := range scimPatchTargets() {
		for _, tc := range cases {
			value, applies := tc.value[target.name]
			if !applies {
				continue
			}
			t.Run(target.name+" "+tc.name, func(t *testing.T) {
				h := newScimHarness()
				tokenStr := issueAllScimToken(t, h.apiTokens)
				resourcePath := target.create(t, h, tokenStr)
				_, before := doScimGet(t, h.echo, tokenStr, resourcePath)

				rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, resourcePath, pathlessPatchOp(tc.op, value))
				if rec.Code != http.StatusBadRequest || body["scimType"] != tc.wantScimType {
					t.Fatalf("expected 400 %s, got %d body=%v", tc.wantScimType, rec.Code, body)
				}
				assertScimResourceUnchanged(t, h, tokenStr, resourcePath, before)
			})
		}
	}
}
