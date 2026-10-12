package handlers_http_test

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// storedScimUser は SCIM id が指す内部 User を保存先から読む。
func storedScimUser(t *testing.T, h scimHarness, scimID string) *userdomain.User {
	t.Helper()
	ctx := context.Background()
	ref, err := h.scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimID)
	if err != nil || ref == nil {
		t.Fatalf("find scim reference %s: %v (%v)", scimID, ref, err)
	}
	user, err := h.userRepo.FindBySub(ctx, ref.UserID)
	if err != nil || user == nil {
		t.Fatalf("find user %s: %v (%v)", ref.UserID, user, err)
	}
	return user
}

func storedEmail(user *userdomain.User) string {
	if user.Email == nil {
		return ""
	}
	return *user.Email
}

func storedStringAttribute(user *userdomain.User, key string) string {
	value, ok := user.Attributes[key]
	if !ok || value.String == nil {
		return ""
	}
	return *value.String
}

// scimUserWrite は CreateScimUser・UpdateScimUser・PatchScimUser の 3 経路へ同じ入力を
// 送るための記述である。書き込みの効果は経路ごとに別の関数を通るので、3 経路とも読む。
type scimUserWrite struct {
	name string
	// replacesExisting は、書き込みの前に対象の User を作っておく経路 (PUT と PATCH) で真である。
	replacesExisting bool
	// send は attrs を書き込む。POST では scimID を使わない。
	send func(t *testing.T, h scimHarness, tokenStr, scimID, userName string, attrs map[string]any) *httptest.ResponseRecorder
}

func scimUserWrites() []scimUserWrite {
	withUserName := func(userName string, attrs map[string]any) map[string]any {
		body := map[string]any{"userName": userName}
		maps.Copy(body, attrs)
		return body
	}
	return []scimUserWrite{
		{name: "POST", send: func(t *testing.T, h scimHarness, tokenStr, _, userName string, attrs map[string]any) *httptest.ResponseRecorder {
			t.Helper()
			rec, _ := doScimJSON(t, h.echo, http.MethodPost, tokenStr, "/scim/v2/Users", withUserName(userName, attrs))
			return rec
		}},
		{name: "PUT", replacesExisting: true, send: func(t *testing.T, h scimHarness, tokenStr, scimID, userName string, attrs map[string]any) *httptest.ResponseRecorder {
			t.Helper()
			rec, _ := doScimJSON(t, h.echo, http.MethodPut, tokenStr, "/scim/v2/Users/"+scimID, withUserName(userName, attrs))
			return rec
		}},
		{name: "PATCH", replacesExisting: true, send: func(t *testing.T, h scimHarness, tokenStr, scimID, _ string, attrs map[string]any) *httptest.ResponseRecorder {
			t.Helper()
			operations := make([]map[string]any, 0, len(attrs))
			for path, value := range patchPaths(attrs) {
				operations = append(operations, map[string]any{"op": "replace", "path": path, "value": value})
			}
			rec, _ := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID, map[string]any{
				"schemas":    []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
				"Operations": operations,
			})
			return rec
		}},
	}
}

// writeScimUser は、必要なら対象の User を先に作ってから attrs を書き込み、状態符号、
// 応答、書き込み先の SCIM id を返す。
func writeScimUser(t *testing.T, h scimHarness, tokenStr string, write scimUserWrite, userName string, attrs map[string]any) (int, map[string]any, string) {
	t.Helper()
	var scimID string
	if write.replacesExisting {
		scimID = createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": userName})
	}
	rec := write.send(t, h, tokenStr, scimID, userName, attrs)
	body := decodeScimBody(t, rec.Body.Bytes())
	if !write.replacesExisting {
		scimID, _ = body["id"].(string)
	}
	return rec.Code, body, scimID
}

// patchPaths は POST/PUT の本文の形を PATCH のパスへ展開する。Enterprise 拡張は
// 拡張オブジェクトの中の属性ごとに URN 修飾のパスになる。
func patchPaths(attrs map[string]any) map[string]any {
	paths := map[string]any{}
	for key, value := range attrs {
		extension, ok := value.(map[string]any)
		if key != enterpriseSchemaURN || !ok {
			paths[key] = value
			continue
		}
		for attr, attrValue := range extension {
			paths[enterpriseSchemaURN+":"+attr] = attrValue
		}
	}
	return paths
}

//spec:covers EX-SOURCING-006-01: 3 つの書き込み経路で primary=true の要素だけを User.email へ保存し、応答が type=work・primary=true の要素を 1 つだけ返し、メールアドレスのない User の応答は emails を持たない。
func TestScimUserEmails_StoresThePrimaryValueAndReturnsOneCanonicalElement(t *testing.T) {
	for _, write := range scimUserWrites() {
		t.Run(write.name, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			status, body, scimID := writeScimUser(t, h, tokenStr, write, "multi-email@example.com", map[string]any{
				"emails": []any{
					map[string]any{"value": "home@example.com", "type": "home"},
					map[string]any{"value": "chosen@example.com", "type": "other", "primary": true},
					map[string]any{"value": "work@example.com", "type": "work"},
				},
			})
			if status != http.StatusOK && status != http.StatusCreated {
				t.Fatalf("expected success, got %d body=%v", status, body)
			}
			if got := storedEmail(storedScimUser(t, h, scimID)); got != "chosen@example.com" {
				t.Errorf("stored email = %q, want chosen@example.com", got)
			}
			emails, _ := body["emails"].([]any)
			if len(emails) != 1 {
				t.Fatalf("response emails = %v, want exactly one element", body["emails"])
			}
			if email, _ := emails[0].(map[string]any); email["value"] != "chosen@example.com" || email["type"] != "work" || email["primary"] != true {
				t.Errorf("response email = %v, want chosen@example.com as type=work primary=true", email)
			}

			withoutEmail := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "no-email@example.com"})
			_, read := doScimGet(t, h.echo, tokenStr, "/scim/v2/Users/"+withoutEmail)
			if _, exists := read["emails"]; exists {
				t.Errorf("emails = %v, want the attribute omitted without a stored address", read["emails"])
			}
		})
	}
}

// 受け付けない側を続けて読む。phoneNumbers と addresses は黙って捨てず拒否し、
// /Schemas はそれらも Group メンバーも広告しない。
//
//spec:covers EX-SOURCING-006-01: phoneNumbers と addresses を POST・PUT では invalidValue、PATCH のパスでは invalidPath で拒否し、/Schemas が emails と User を指すメンバーだけを広告して phoneNumbers・addresses・Group メンバーを広告しない。
func TestScimUserEmails_AdvertisesOnlyTheSupportedProjection(t *testing.T) {
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)

	for _, attr := range []string{"phoneNumbers", "addresses"} {
		for _, write := range scimUserWrites() {
			t.Run(write.name+" "+attr, func(t *testing.T) {
				status, body, _ := writeScimUser(t, h, tokenStr, write, write.name+"-"+attr+"@example.com", map[string]any{attr: []any{}})
				want := "invalidValue"
				if write.name == "PATCH" {
					want = "invalidPath"
				}
				if status != http.StatusBadRequest || body["scimType"] != want {
					t.Errorf("expected 400 %s, got %d body=%v", want, status, body)
				}
			})
		}
	}

	rec := doScimRequest(t, h.echo, http.MethodGet, "Bearer "+tokenStr, "/scim/v2/Schemas", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var schemas []scimSchemaDocument
	if err := json.Unmarshal(rec.Body.Bytes(), &schemas); err != nil {
		t.Fatalf("failed to decode schemas: %v", err)
	}
	user := findScimSchema(t, schemas, "urn:ietf:params:scim:schemas:core:2.0:User")
	if _, ok := user.attribute("emails"); !ok {
		t.Errorf("User schema attributes = %v, want emails", user.names())
	}
	for _, unsupported := range []string{"phoneNumbers", "addresses"} {
		if _, ok := user.attribute(unsupported); ok {
			t.Errorf("User schema advertises %s", unsupported)
		}
	}
	group := findScimSchema(t, schemas, "urn:ietf:params:scim:schemas:core:2.0:Group")
	members, ok := group.attribute("members")
	if !ok {
		t.Fatalf("Group schema attributes = %v, want members", group.names())
	}
	for _, sub := range members.SubAttributes {
		if sub.Name == "$ref" && !slices.Equal(sub.ReferenceTypes, []string{"User"}) {
			t.Errorf("members.$ref referenceTypes = %v, want [User]", sub.ReferenceTypes)
		}
	}
}

type scimSchemaAttribute struct {
	Name           string                `json:"name"`
	ReferenceTypes []string              `json:"referenceTypes"`
	SubAttributes  []scimSchemaAttribute `json:"subAttributes"`
}

type scimSchemaDocument struct {
	ID         string                `json:"id"`
	Attributes []scimSchemaAttribute `json:"attributes"`
}

func (d scimSchemaDocument) attribute(name string) (scimSchemaAttribute, bool) {
	for _, attr := range d.Attributes {
		if attr.Name == name {
			return attr, true
		}
	}
	return scimSchemaAttribute{}, false
}

func (d scimSchemaDocument) names() []string {
	names := make([]string, 0, len(d.Attributes))
	for _, attr := range d.Attributes {
		names = append(names, attr.Name)
	}
	return names
}

func findScimSchema(t *testing.T, schemas []scimSchemaDocument, id string) scimSchemaDocument {
	t.Helper()
	for _, schema := range schemas {
		if schema.ID == id {
			return schema
		}
	}
	t.Fatalf("schema %s is not advertised", id)
	return scimSchemaDocument{}
}

// POST は作成されないこと、PUT と PATCH は読み直した User が拒否の前と同じであることを読む。
//
//spec:covers EX-SOURCING-006-02: 複数の primary=true、オブジェクトでない要素、空または文字列でない value、文字列でない type、真偽値でない primary の emails を 3 つの書き込み経路で 400 invalidValue と拒否し、User を作らず変えない。
func TestScimUserEmails_RefusesMalformedElementsAndLeavesTheUserUnchanged(t *testing.T) {
	malformed := map[string][]any{
		"two primaries":       {map[string]any{"value": "a@example.com", "primary": true}, map[string]any{"value": "b@example.com", "primary": true}},
		"element not object":  {"a@example.com"},
		"empty value":         {map[string]any{"value": ""}},
		"value not string":    {map[string]any{"value": 42}},
		"type not string":     {map[string]any{"value": "a@example.com", "type": 42}},
		"primary not boolean": {map[string]any{"value": "a@example.com", "primary": "true"}},
	}
	for name, emails := range malformed {
		for _, write := range scimUserWrites() {
			t.Run(write.name+" "+name, func(t *testing.T) {
				assertScimUserWriteRefused(t, write, map[string]any{"emails": emails})
			})
		}
	}
}

//spec:covers EX-SOURCING-006-03: primary のない emails では、3 つの書き込み経路とも type が大文字小文字を問わず work に一致する最初の要素を User.email へ保存する。
//spec:covers EX-SOURCING-006-04: primary も work もない emails では、3 つの書き込み経路とも通信上で最初の要素を User.email へ保存する。
func TestScimUserEmails_FallsBackToWorkThenWireOrder(t *testing.T) {
	cases := []struct {
		name   string
		emails []any
		want   string
	}{
		{name: "work", emails: []any{
			map[string]any{"value": "home@example.com", "type": "home"},
			map[string]any{"value": "first-work@example.com", "type": "WoRk"},
			map[string]any{"value": "second-work@example.com", "type": "work"},
		}, want: "first-work@example.com"},
		{name: "wire order", emails: []any{
			map[string]any{"value": "first@example.com", "type": "home"},
			map[string]any{"value": "second@example.com"},
		}, want: "first@example.com"},
	}
	for _, tc := range cases {
		for _, write := range scimUserWrites() {
			t.Run(write.name+" "+tc.name, func(t *testing.T) {
				h := newScimHarness()
				tokenStr := issueAllScimToken(t, h.apiTokens)
				status, body, scimID := writeScimUser(t, h, tokenStr, write, "fallback@example.com", map[string]any{"emails": tc.emails})
				if status != http.StatusOK && status != http.StatusCreated {
					t.Fatalf("expected success, got %d body=%v", status, body)
				}
				if got := storedEmail(storedScimUser(t, h, scimID)); got != tc.want {
					t.Errorf("stored email = %q, want %q", got, tc.want)
				}
			})
		}
	}
}

// manager は SCIM id で届くが、保存するのは解決した内部 User の sub である。
//
//spec:covers EX-SOURCING-007-01: 3 つの書き込み経路で employeeNumber・department・manager を User.Attributes の employee_number・department・manager_sub へ保存し、manager_sub が解決した内部 User の sub であり、応答の schemas が Enterprise 拡張 URN を含む。
func TestScimEnterpriseExtension_PersistsTheOrganizationAttributes(t *testing.T) {
	for _, write := range scimUserWrites() {
		t.Run(write.name, func(t *testing.T) {
			h := newScimHarness()
			tokenStr := issueAllScimToken(t, h.apiTokens)
			managerScimID := createScimUserForTest(t, h.echo, tokenStr, map[string]any{"userName": "boss@example.com"})
			managerSub := storedScimUser(t, h, managerScimID).ID

			status, body, scimID := writeScimUser(t, h, tokenStr, write, "report@example.com", map[string]any{
				enterpriseSchemaURN: map[string]any{
					"employeeNumber": "E-7",
					"department":     "Research",
					"manager":        map[string]any{"value": managerScimID},
				},
			})
			if status != http.StatusOK && status != http.StatusCreated {
				t.Fatalf("expected success, got %d body=%v", status, body)
			}
			stored := storedScimUser(t, h, scimID)
			for key, want := range map[string]string{"employee_number": "E-7", "department": "Research", "manager_sub": managerSub} {
				if got := storedStringAttribute(stored, key); got != want {
					t.Errorf("Attributes[%s] = %q, want %q", key, got, want)
				}
			}
			schemas, _ := body["schemas"].([]any)
			if !slices.Contains(schemas, any(enterpriseSchemaURN)) {
				t.Errorf("schemas = %v, want the enterprise extension URN", schemas)
			}
		})
	}
}

//spec:covers EX-SOURCING-007-02: 文字列でない employeeNumber と department を 3 つの書き込み経路で 400 invalidValue と拒否し、User を作らず変えない。
func TestScimEnterpriseExtension_RefusesANonStringAttributeAndLeavesTheUserUnchanged(t *testing.T) {
	invalid := map[string]map[string]any{
		"employeeNumber not string": {"employeeNumber": 42},
		"department not string":     {"department": true},
	}
	for name, extension := range invalid {
		for _, write := range scimUserWrites() {
			t.Run(write.name+" "+name, func(t *testing.T) {
				assertScimUserWriteRefused(t, write, map[string]any{enterpriseSchemaURN: extension})
			})
		}
	}
}

// manager の解決は保存先を引くので、本文の形の検証より後に失敗する。PUT は他の属性を、
// PATCH は先行する操作を書き換えた後でこの失敗に出会いうるので、どちらも拒否の前後で
// User の表現全体を比べる。
//
//spec:covers EX-SOURCING-007-03: 空の value オブジェクト、空文字、テナントに存在しない SCIM User を指す manager を 3 つの書き込み経路で 400 invalidValue と拒否し、PATCH で先行する操作があっても User を作らず変えない。
func TestScimEnterpriseExtension_RefusesAnUnresolvableManagerAndLeavesTheUserUnchanged(t *testing.T) {
	invalid := map[string]map[string]any{
		"empty object": {"manager": map[string]any{}},
		"empty value":  {"manager": map[string]any{"value": ""}},
		"empty string": {"manager": ""},
		"unknown":      {"manager": map[string]any{"value": "does-not-exist"}},
	}
	for name, extension := range invalid {
		for _, write := range scimUserWrites() {
			t.Run(write.name+" "+name, func(t *testing.T) {
				assertScimUserWriteRefused(t, write, map[string]any{enterpriseSchemaURN: extension})
			})
		}
	}

	t.Run("PATCH unknown after another operation", func(t *testing.T) {
		h := newScimHarness()
		tokenStr := issueAllScimToken(t, h.apiTokens)
		userPath := "/scim/v2/Users/" + createScimUserForTest(t, h.echo, tokenStr, map[string]any{
			"userName": "ordered@example.com",
			"emails":   []any{map[string]any{"value": "kept@example.com", "primary": true}},
		})
		_, before := doScimGet(t, h.echo, tokenStr, userPath)

		rec, body := doScimJSON(t, h.echo, http.MethodPatch, tokenStr, userPath, map[string]any{
			"schemas": []string{"urn:ietf:params:scim:api:messages:2.0:PatchOp"},
			"Operations": []map[string]any{
				{"op": "replace", "path": "active", "value": false},
				{"op": "remove", "path": "emails"},
				{"op": "replace", "path": "manager", "value": "does-not-exist"},
			},
		})
		if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
			t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
		}
		assertScimResourceUnchanged(t, h, tokenStr, userPath, before)
	})
}

// assertScimUserWriteRefused は attrs の書き込みが 400 invalidValue で拒否され、POST なら
// User が作られず、PUT と PATCH なら読み直した User が拒否の前と同じであることを読む。
func assertScimUserWriteRefused(t *testing.T, write scimUserWrite, attrs map[string]any) {
	t.Helper()
	ctx := context.Background()
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)
	const userName = "refused@example.com"

	var scimID string
	var before map[string]any
	if write.replacesExisting {
		scimID = createScimUserForTest(t, h.echo, tokenStr, map[string]any{
			"userName":          userName,
			"emails":            []any{map[string]any{"value": "kept@example.com", "primary": true}},
			enterpriseSchemaURN: map[string]any{"employeeNumber": "E-1", "department": "Kept"},
		})
		_, before = doScimGet(t, h.echo, tokenStr, "/scim/v2/Users/"+scimID)
	}

	rec := write.send(t, h, tokenStr, scimID, userName, attrs)
	body := decodeScimBody(t, rec.Body.Bytes())
	if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidValue" {
		t.Fatalf("expected 400 invalidValue, got %d body=%v", rec.Code, body)
	}
	if !write.replacesExisting {
		if count, err := h.userRepo.Count(ctx, tenancydomain.DefaultTenantID); err != nil || count != 0 {
			t.Errorf("user count = %d (%v), want 0 because the refused create stored nothing", count, err)
		}
		return
	}
	assertScimResourceUnchanged(t, h, tokenStr, "/scim/v2/Users/"+scimID, before)
}
