package handlers_http_test

// docs/contexts/sourcing/standards.md が宣言する採用規範のうち、SCIM の
// サービス提供者としての入口 (/scim/v2/...) から観測できる行を固定する。

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/labstack/echo/v5"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	scimErrorSchemaURN  = "urn:ietf:params:scim:api:messages:2.0:Error"
	enterpriseSchemaURN = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
)

// doScimRequest は Authorization header を呼び出し側が組み立てる要求を送る。
// doScimGet / doScimJSON は正しい Bearer 前提なので、header そのものを崩す
// 観測には使えない。
func doScimRequest(t *testing.T, e *echo.Echo, method, authorization, path string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	var payload io.Reader = http.NoBody
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, payload)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	req.Header.Set("Content-Type", "application/scim+json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// 持ち、その中で Bearer トークン方式を広告することを固定する。SCIM クライアントは
// ここを読んで認証方式を選ぶので、空の配列や別方式だけの広告と区別する。
//
//spec:covers RFC7643-SERVICE-PROVIDER-CONFIG: ServiceProviderConfig が authenticationSchemes を
func TestScimServiceProviderConfig_AdvertisesTheBearerAuthenticationScheme(t *testing.T) {
	h := newScimHarness()
	tokenStr := issueAllScimToken(t, h.apiTokens)

	rec, body := doScimGet(t, h.echo, tokenStr, "/scim/v2/ServiceProviderConfig")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	schemes, _ := body["authenticationSchemes"].([]any)
	if len(schemes) == 0 {
		t.Fatalf("expected a non-empty authenticationSchemes, got %v", body["authenticationSchemes"])
	}

	var bearer map[string]any
	for _, raw := range schemes {
		scheme, _ := raw.(map[string]any)
		if scheme["type"] == "oauthbearertoken" {
			bearer = scheme
		}
	}
	if bearer == nil {
		t.Fatalf("expected an oauthbearertoken scheme, got %v", schemes)
	}
	if name, _ := bearer["name"].(string); name == "" {
		t.Errorf("expected the bearer scheme to carry a name, got %v", bearer)
	}
}

// 4 操作が SCIM の入口から到達できることを固定する。置換と削除は、応答だけでなく
// 後続の参照でも効果を読み、応答の組み立てで終わっている実装と区別する。
//
//spec:covers RFC7644-RESOURCE-OPERATIONS: User と Group のどちらにも、作成・参照・置換・削除の
func TestScimResourceOperations_UsersAndGroupsSupportCreateReadReplaceDelete(t *testing.T) {
	h := newScimHarness()
	e := h.echo
	ctx := context.Background()
	tokenStr := issueAllScimToken(t, h.apiTokens)

	t.Run("User", func(t *testing.T) {
		createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "lifecycle@example.com",
			"name":     map[string]any{"givenName": "Life", "familyName": "Cycle"},
		})
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create: expected 201, got %d body=%v", createRec.Code, created)
		}
		scimID, _ := created["id"].(string)
		if scimID == "" {
			t.Fatalf("create: expected a server-assigned id, got %v", created)
		}

		getRec, got := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+scimID)
		if getRec.Code != http.StatusOK {
			t.Fatalf("read: expected 200, got %d body=%v", getRec.Code, got)
		}
		if got["id"] != scimID || got["userName"] != "lifecycle@example.com" {
			t.Errorf("read: got %v, want the created resource", got)
		}

		putRec, replaced := doScimJSON(t, e, http.MethodPut, tokenStr, "/scim/v2/Users/"+scimID, map[string]any{
			"userName": "replaced@example.com",
		})
		if putRec.Code != http.StatusOK {
			t.Fatalf("replace: expected 200, got %d body=%v", putRec.Code, replaced)
		}
		if replaced["userName"] != "replaced@example.com" {
			t.Errorf("replace: userName = %v, want replaced@example.com", replaced["userName"])
		}
		_, afterReplace := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+scimID)
		if afterReplace["userName"] != "replaced@example.com" {
			t.Errorf("replace: a re-read returned %v, so the replacement was not persisted", afterReplace["userName"])
		}

		deleteRec := doScimRequest(t, e, http.MethodDelete, "Bearer "+tokenStr, "/scim/v2/Users/"+scimID, nil)
		if deleteRec.Code != http.StatusNoContent {
			t.Fatalf("delete: expected 204, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
		}
		// User の削除は soft delete なので、効果は SCIM の表現ではなく
		// User の lifecycle に現れる。
		ref, err := h.scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimID)
		if err != nil || ref == nil {
			t.Fatalf("delete: expected the scim reference to remain, got %v (%v)", ref, err)
		}
		user, err := h.userRepo.FindBySub(ctx, ref.UserID)
		if err != nil || user == nil {
			t.Fatalf("delete: expected the user to remain readable, got %v (%v)", user, err)
		}
		if user.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
			t.Errorf("delete: status = %s, want %s", user.Lifecycle.Status, idmdomain.UserStatusPendingDeletion)
		}
	})

	t.Run("Group", func(t *testing.T) {
		createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "Lifecycle",
		})
		if createRec.Code != http.StatusCreated {
			t.Fatalf("create: expected 201, got %d body=%v", createRec.Code, created)
		}
		scimID, _ := created["id"].(string)
		if scimID == "" {
			t.Fatalf("create: expected a server-assigned id, got %v", created)
		}

		getRec, got := doScimGet(t, e, tokenStr, "/scim/v2/Groups/"+scimID)
		if getRec.Code != http.StatusOK {
			t.Fatalf("read: expected 200, got %d body=%v", getRec.Code, got)
		}
		if got["id"] != scimID || got["displayName"] != "Lifecycle" {
			t.Errorf("read: got %v, want the created resource", got)
		}

		putRec, replaced := doScimJSON(t, e, http.MethodPut, tokenStr, "/scim/v2/Groups/"+scimID, map[string]any{
			"displayName": "Lifecycle Renamed",
		})
		if putRec.Code != http.StatusOK {
			t.Fatalf("replace: expected 200, got %d body=%v", putRec.Code, replaced)
		}
		if replaced["displayName"] != "Lifecycle Renamed" {
			t.Errorf("replace: displayName = %v, want Lifecycle Renamed", replaced["displayName"])
		}
		_, afterReplace := doScimGet(t, e, tokenStr, "/scim/v2/Groups/"+scimID)
		if afterReplace["displayName"] != "Lifecycle Renamed" {
			t.Errorf("replace: a re-read returned %v, so the replacement was not persisted", afterReplace["displayName"])
		}

		deleteRec := doScimRequest(t, e, http.MethodDelete, "Bearer "+tokenStr, "/scim/v2/Groups/"+scimID, nil)
		if deleteRec.Code != http.StatusNoContent {
			t.Fatalf("delete: expected 204, got %d body=%s", deleteRec.Code, deleteRec.Body.String())
		}
		afterDeleteRec, afterDelete := doScimGet(t, e, tokenStr, "/scim/v2/Groups/"+scimID)
		if afterDeleteRec.Code != http.StatusNotFound {
			t.Errorf("delete: a re-read returned %d body=%v, so the group was not removed", afterDeleteRec.Code, afterDelete)
		}
	})
}

// API アクセストークンを要求する。拒否そのものと、**その拒否が防いだ効果**、
// つまり対象リソースが変わっていないことを対で固定する。状態符号だけでは、
// 変更してから拒否を返す実装と区別できない。
//
//spec:covers RFC7644-BEARER-AUTHORIZATION: SCIM の入口は、該当スコープを持つテナント単位の
func TestScimBearerAuthorization_RefusesAndLeavesTheResourceUnchanged(t *testing.T) {
	h := newScimHarness()
	e := h.echo
	ctx := context.Background()
	tokenStr := issueAllScimToken(t, h.apiTokens)

	createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
		"userName": "guarded@example.com",
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d body=%v", createRec.Code, created)
	}
	scimID := created["id"].(string)

	readOnly, _, err := h.apiTokens.Issue(ctx, tenancydomain.DefaultTenantID, "admin-test", "read only",
		[]string{string(apitokendomain.ScopeScimUsersRead)}, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	otherTenant, _, err := h.apiTokens.Issue(ctx, "other-tenant", "admin-test", "other tenant",
		[]string{string(apitokendomain.ScopeScimUsersRead), string(apitokendomain.ScopeScimUsersWrite)}, 1, "")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name          string
		authorization string
		wantStatus    int
	}{
		{name: "no Authorization header", authorization: "", wantStatus: http.StatusUnauthorized},
		{
			name:          "credentials outside the Bearer scheme",
			authorization: "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret")),
			wantStatus:    http.StatusUnauthorized,
		},
		{name: "a token this server never issued", authorization: "Bearer never-issued-token", wantStatus: http.StatusUnauthorized},
		{name: "a token issued to another tenant", authorization: "Bearer " + otherTenant, wantStatus: http.StatusUnauthorized},
		{name: "a token without the write scope", authorization: "Bearer " + readOnly, wantStatus: http.StatusForbidden},
	}

	attempts := []struct {
		what   string
		method string
		body   map[string]any
	}{
		{what: "replace", method: http.MethodPut, body: map[string]any{"userName": "taken-over@example.com"}},
		{what: "disable", method: http.MethodPatch, body: patchOp("replace", "active", false)},
		{what: "delete", method: http.MethodDelete},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, attempt := range attempts {
				rec := doScimRequest(t, e, attempt.method, tc.authorization, "/scim/v2/Users/"+scimID, attempt.body)
				if rec.Code != tc.wantStatus {
					t.Errorf("%s: status = %d, want %d body=%s", attempt.what, rec.Code, tc.wantStatus, rec.Body.String())
				}
				if challenge := rec.Header().Get("WWW-Authenticate"); challenge == "" {
					t.Errorf("%s: expected a WWW-Authenticate challenge", attempt.what)
				}
			}

			// 拒否が防いだ効果。要求した 3 つの変更 (置換・無効化・削除) は
			// どれもリソースに届いていない。
			rec, after := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+scimID)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected the user to remain readable, got %d body=%v", rec.Code, after)
			}
			if after["userName"] != "guarded@example.com" {
				t.Errorf("userName = %v, want guarded@example.com", after["userName"])
			}
			if after["active"] != true {
				t.Errorf("active = %v, want true", after["active"])
			}
		})
	}
}

// schemas / status / detail と application/scim+json まで読み、状態符号だけが
// 合っている素の JSON と区別する。
//
//spec:covers RFC7644-ERROR-RESPONSE: プロトコル上の失敗は、SCIM の誤り応答の形で返る。
func TestScimErrorResponse_FailuresCarryTheScimErrorBody(t *testing.T) {
	h := newScimHarness()
	e := h.echo
	ctx := context.Background()
	tokenStr := issueAllScimToken(t, h.apiTokens)

	readOnly, _, err := h.apiTokens.Issue(ctx, tenancydomain.DefaultTenantID, "admin-test", "read only",
		[]string{string(apitokendomain.ScopeScimUsersRead)}, 1, "")
	if err != nil {
		t.Fatal(err)
	}

	createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
		"userName": "occupied@example.com",
	})
	if createRec.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d body=%v", createRec.Code, created)
	}
	scimID := created["id"].(string)

	cases := []struct {
		name          string
		method        string
		authorization string
		path          string
		body          map[string]any
		wantStatus    int
		wantScimType  string
	}{
		{
			name: "a token this server never issued", method: http.MethodGet,
			authorization: "Bearer never-issued-token", path: "/scim/v2/Users",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "a token without the write scope", method: http.MethodPost,
			authorization: "Bearer " + readOnly, path: "/scim/v2/Users",
			body: map[string]any{"userName": "rejected@example.com"}, wantStatus: http.StatusForbidden,
		},
		{
			name: "a missing required attribute", method: http.MethodPost,
			authorization: "Bearer " + tokenStr, path: "/scim/v2/Users",
			body: map[string]any{}, wantStatus: http.StatusBadRequest, wantScimType: "invalidValue",
		},
		{
			name: "an unsupported PATCH path", method: http.MethodPatch,
			authorization: "Bearer " + tokenStr, path: "/scim/v2/Users/" + scimID,
			body: patchOp("replace", "nickName", "x"), wantStatus: http.StatusBadRequest, wantScimType: "invalidPath",
		},
		{
			name: "an unparsable filter", method: http.MethodGet,
			authorization: "Bearer " + tokenStr,
			path:          "/scim/v2/Users?filter=" + url.QueryEscape(`userName eq`),
			wantStatus:    http.StatusBadRequest, wantScimType: "invalidFilter",
		},
		{
			name: "an unknown resource", method: http.MethodGet,
			authorization: "Bearer " + tokenStr, path: "/scim/v2/Users/does-not-exist",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "a duplicate userName", method: http.MethodPost,
			authorization: "Bearer " + tokenStr, path: "/scim/v2/Users",
			body: map[string]any{"userName": "occupied@example.com"}, wantStatus: http.StatusConflict,
			wantScimType: "uniqueness",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doScimRequest(t, e, tc.method, tc.authorization, tc.path, tc.body)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d body=%s", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if contentType := rec.Header().Get("Content-Type"); contentType != "application/scim+json" {
				t.Errorf("Content-Type = %q, want application/scim+json", contentType)
			}

			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode the error body %q: %v", rec.Body.String(), err)
			}
			schemas, _ := body["schemas"].([]any)
			if len(schemas) != 1 || schemas[0] != scimErrorSchemaURN {
				t.Errorf("schemas = %v, want [%s]", body["schemas"], scimErrorSchemaURN)
			}
			if body["status"] != strconv.Itoa(tc.wantStatus) {
				t.Errorf("status = %v, want %q", body["status"], strconv.Itoa(tc.wantStatus))
			}
			if detail, _ := body["detail"].(string); detail == "" {
				t.Errorf("expected a non-empty detail, got %v", body)
			}
			if tc.wantScimType != "" && body["scimType"] != tc.wantScimType {
				t.Errorf("scimType = %v, want %s", body["scimType"], tc.wantScimType)
			}
		})
	}
}

// employeeNumber / department / manager の 3 属性だけである。採用した側
// (Discovery と CRUD / PATCH で扱う) と、採用していない側 (広告せず、作成では
// 保存も応答もせず、PATCH では拒否する) を続けて観測する。片側だけでは、
// 拡張を全部採用している実装とも、何も採用していない実装とも区別できない。
//
//spec:covers RFC7643-ENTERPRISE-EXTENSION: 採用したのは Enterprise 拡張のうち
func TestScimEnterpriseExtension_AdoptsOnlyTheDeclaredSubset(t *testing.T) {
	h := newScimHarness()
	e := h.echo
	tokenStr := issueAllScimToken(t, h.apiTokens)
	unadopted := []string{"costCenter", "division", "organization"}

	t.Run("the adopted attributes are stored and returned", func(t *testing.T) {
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "adopted@example.com",
			enterpriseSchemaURN: map[string]any{
				"employeeNumber": "E-1",
				"department":     "Tour Operations",
			},
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		getRec, got := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+body["id"].(string))
		if getRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", getRec.Code, got)
		}
		ext, _ := got[enterpriseSchemaURN].(map[string]any)
		if ext["employeeNumber"] != "E-1" || ext["department"] != "Tour Operations" {
			t.Errorf("enterprise extension = %v, want the submitted employeeNumber and department", got[enterpriseSchemaURN])
		}
	})

	t.Run("an unadopted attribute is neither stored nor returned", func(t *testing.T) {
		submitted := map[string]any{"employeeNumber": "E-2"}
		for _, attr := range unadopted {
			submitted[attr] = "submitted-" + attr
		}
		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName":          "unadopted@example.com",
			enterpriseSchemaURN: submitted,
		})
		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d body=%v", rec.Code, body)
		}
		getRec, got := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+body["id"].(string))
		if getRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", getRec.Code, got)
		}
		ext, _ := got[enterpriseSchemaURN].(map[string]any)
		if ext["employeeNumber"] != "E-2" {
			t.Fatalf("expected the adopted attribute to survive, got %v", got[enterpriseSchemaURN])
		}
		for _, attr := range unadopted {
			if _, exists := ext[attr]; exists {
				t.Errorf("%s = %v, want the unadopted attribute to be absent", attr, ext[attr])
			}
		}
	})

	t.Run("an unadopted attribute is refused by PATCH", func(t *testing.T) {
		createRec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "patch-boundary@example.com",
		})
		if createRec.Code != http.StatusCreated {
			t.Fatalf("setup create failed: %d body=%v", createRec.Code, created)
		}
		scimID := created["id"].(string)

		for _, attr := range unadopted {
			for _, path := range []string{attr, enterpriseSchemaURN + ":" + attr} {
				rec, body := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
					patchOp("replace", path, "x"))
				if rec.Code != http.StatusBadRequest || body["scimType"] != "invalidPath" {
					t.Errorf("PATCH %q: got %d body=%v, want 400 invalidPath", path, rec.Code, body)
				}
			}
		}
	})

	t.Run("Discovery advertises exactly the adopted attributes", func(t *testing.T) {
		rec := doScimRequest(t, e, http.MethodGet, "Bearer "+tokenStr, "/scim/v2/Schemas", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
		}
		var schemas []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &schemas); err != nil {
			t.Fatalf("failed to decode schemas: %v", err)
		}

		var advertised []string
		found := false
		for _, schema := range schemas {
			if schema["id"] != enterpriseSchemaURN {
				continue
			}
			found = true
			attrs, _ := schema["attributes"].([]any)
			for _, raw := range attrs {
				attr, _ := raw.(map[string]any)
				name, _ := attr["name"].(string)
				advertised = append(advertised, name)
			}
		}
		if !found {
			t.Fatalf("expected the enterprise extension schema in %v", schemas)
		}
		if len(advertised) != 3 {
			t.Fatalf("advertised attributes = %v, want exactly employeeNumber, department, manager", advertised)
		}
		wanted := map[string]bool{"employeeNumber": true, "department": true, "manager": true}
		for _, name := range advertised {
			if !wanted[name] {
				t.Errorf("advertised attributes = %v, want exactly employeeNumber, department, manager", advertised)
			}
		}
	})
}

// createScimUserForTest は削除の意味論を観測する前段で User を 1 件作り、その
// SCIM id を返す。作成そのものは別の行が固定しているので、ここでは失敗を
// setup の失敗として扱う。
func createScimUserForTest(t *testing.T, e *echo.Echo, tokenStr string, body map[string]any) string {
	t.Helper()
	rec, created := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("setup: create user: expected 201, got %d body=%v", rec.Code, created)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("setup: create user: expected a server-assigned id, got %v", created)
	}
	return id
}

// deleteScimUserForTest は User を SCIM の DELETE で削除する。
func deleteScimUserForTest(t *testing.T, e *echo.Echo, tokenStr, scimID string) {
	t.Helper()
	rec := doScimRequest(t, e, http.MethodDelete, "Bearer "+tokenStr, "/scim/v2/Users/"+scimID, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("setup: delete user: expected 204, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// scimResourceIDs は ListResponse の Resources から id だけを取り出す。
func scimResourceIDs(t *testing.T, body map[string]any) []string {
	t.Helper()
	resources, _ := body["Resources"].([]any)
	ids := make([]string, 0, len(resources))
	for _, raw := range resources {
		resource, _ := raw.(map[string]any)
		id, _ := resource["id"].(string)
		ids = append(ids, id)
	}
	return ids
}

// 消える。内部では soft delete なのでレコードは残るが、SCIM クライアントから見て
// 削除済みの id は「存在しない id」と区別できてはいけない。
//
// 採用した側と採用していない側の両方を観測する。採用した側は、以後の 4 操作が
// 404 になること、コレクションの照会結果に出ないこと、Group の members と他の
// User の manager にも出ないこと。採用していない側は、同じ userName での再作成が
// 409 uniqueness になること (§3.6 の SHOULD を採っていない)。
//
// 無効化した User が消えないことも対で固定する。判定を「Active でない」と書くと
// 無効化まで消え、外部 IdP が無効化を削除と読む。
//
//spec:covers RFC7644-DELETE-SEMANTICS, EX-SOURCING-002-05: 削除した User は SCIM の表面から
func TestScimDeleteSemantics_DeletedUserIsGoneFromTheScimSurface(t *testing.T) {
	h := newScimHarness()
	e := h.echo
	ctx := context.Background()
	tokenStr := issueAllScimToken(t, h.apiTokens)

	t.Run("every later operation on the deleted id gives 404", func(t *testing.T) {
		scimID := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "leaver@example.com"})
		deleteScimUserForTest(t, e, tokenStr, scimID)

		// 内部レコードは残っている。404 は soft delete をやめた結果ではなく、
		// その選択を SCIM の表現へ反映した結果である。
		ref, err := h.scimRepo.FindUserRefByScimID(ctx, tenancydomain.DefaultTenantID, scimID)
		if err != nil || ref == nil {
			t.Fatalf("expected the scim reference to remain, got %v (%v)", ref, err)
		}
		user, err := h.userRepo.FindBySub(ctx, ref.UserID)
		if err != nil || user == nil {
			t.Fatalf("expected the internal user to remain, got %v (%v)", user, err)
		}
		if user.Lifecycle.Status != idmdomain.UserStatusPendingDeletion {
			t.Fatalf("status = %s, want %s", user.Lifecycle.Status, idmdomain.UserStatusPendingDeletion)
		}

		attempts := []struct {
			what   string
			method string
			body   map[string]any
		}{
			{what: "read", method: http.MethodGet},
			{what: "replace", method: http.MethodPut, body: map[string]any{"userName": "leaver@example.com"}},
			{what: "patch", method: http.MethodPatch, body: patchOp("replace", "active", true)},
			{what: "delete again", method: http.MethodDelete},
		}
		for _, attempt := range attempts {
			t.Run(attempt.what, func(t *testing.T) {
				rec := doScimRequest(t, e, attempt.method, "Bearer "+tokenStr, "/scim/v2/Users/"+scimID, attempt.body)
				if rec.Code != http.StatusNotFound {
					t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
				}
				var body map[string]any
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("expected a SCIM error body, got %q", rec.Body.String())
				}
				schemas, _ := body["schemas"].([]any)
				if len(schemas) == 0 || schemas[0] != scimErrorSchemaURN {
					t.Errorf("schemas = %v, want the SCIM error schema", body["schemas"])
				}
			})
		}
	})

	t.Run("the collection omits it from Resources and totalResults", func(t *testing.T) {
		h := newScimHarness()
		e := h.echo
		tokenStr := issueAllScimToken(t, h.apiTokens)

		kept := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "kept@example.com"})
		removed := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "removed@example.com"})
		deleteScimUserForTest(t, e, tokenStr, removed)

		rec, body := doScimGet(t, e, tokenStr, "/scim/v2/Users")
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", rec.Code, body)
		}
		// totalResults は Resources の長さとは別に組み立てられるので、両方読む。
		// 片方だけでは、除外を射影の後で数えている実装と区別できない。
		if total, _ := body["totalResults"].(float64); int(total) != 1 {
			t.Errorf("totalResults = %v, want 1", body["totalResults"])
		}
		ids := scimResourceIDs(t, body)
		if len(ids) != 1 || ids[0] != kept {
			t.Errorf("Resources ids = %v, want only %q", ids, kept)
		}
	})

	t.Run("a Group neither lists it nor accepts it as a member", func(t *testing.T) {
		h := newScimHarness()
		e := h.echo
		tokenStr := issueAllScimToken(t, h.apiTokens)

		staying := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "staying@example.com"})
		leaving := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "leaving@example.com"})

		createRec, group := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Groups", map[string]any{
			"displayName": "Team",
			"members":     []map[string]any{{"value": staying}, {"value": leaving}},
		})
		if createRec.Code != http.StatusCreated {
			t.Fatalf("setup: create group: expected 201, got %d body=%v", createRec.Code, group)
		}
		groupID := group["id"].(string)

		deleteScimUserForTest(t, e, tokenStr, leaving)

		readRec, read := doScimGet(t, e, tokenStr, "/scim/v2/Groups/"+groupID)
		if readRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", readRec.Code, read)
		}
		members, _ := read["members"].([]any)
		values := make([]string, 0, len(members))
		for _, raw := range members {
			member, _ := raw.(map[string]any)
			value, _ := member["value"].(string)
			values = append(values, value)
		}
		if len(values) != 1 || values[0] != staying {
			t.Errorf("members = %v, want only %q", values, staying)
		}

		// 削除済みの id を member として受け取る側も塞ぐ。拒否そのものと、
		// 拒否が Group を変えなかったことを対で読む。
		patchRec, refused := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Groups/"+groupID,
			patchOp("add", "members", []map[string]any{{"value": leaving}}))
		if patchRec.Code != http.StatusBadRequest {
			t.Fatalf("adding a deleted member: expected 400, got %d body=%v", patchRec.Code, refused)
		}
		if refused["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", refused["scimType"])
		}
		_, afterRefusal := doScimGet(t, e, tokenStr, "/scim/v2/Groups/"+groupID)
		if got := len(afterRefusal["members"].([]any)); got != 1 {
			t.Errorf("after the refusal the group has %d members, want 1", got)
		}
	})

	t.Run("the enterprise manager reference neither appears nor is accepted", func(t *testing.T) {
		h := newScimHarness()
		e := h.echo
		tokenStr := issueAllScimToken(t, h.apiTokens)

		manager := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "manager@example.com"})
		report := createScimUserForTest(t, e, tokenStr, map[string]any{
			"userName": "report@example.com",
			enterpriseSchemaURN: map[string]any{
				"manager": map[string]any{"value": manager},
			},
		})

		deleteScimUserForTest(t, e, tokenStr, manager)

		readRec, read := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+report)
		if readRec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d body=%v", readRec.Code, read)
		}
		// 読み取りが出した参照を書き込みが拒むと read-modify-write が壊れるので、
		// 削除済みを指す manager は応答から消す。
		if ext, ok := read[enterpriseSchemaURN].(map[string]any); ok {
			if _, present := ext["manager"]; present {
				t.Errorf("manager = %v, want the reference to a deleted user to be gone", ext["manager"])
			}
		}

		patchRec, refused := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+report,
			patchOp("replace", "manager", manager))
		if patchRec.Code != http.StatusBadRequest {
			t.Fatalf("naming a deleted manager: expected 400, got %d body=%v", patchRec.Code, refused)
		}
		if refused["scimType"] != "invalidValue" {
			t.Errorf("scimType = %v, want invalidValue", refused["scimType"])
		}
	})

	t.Run("a disabled user stays visible", func(t *testing.T) {
		h := newScimHarness()
		e := h.echo
		tokenStr := issueAllScimToken(t, h.apiTokens)

		scimID := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "onleave@example.com"})
		patchRec, patched := doScimJSON(t, e, http.MethodPatch, tokenStr, "/scim/v2/Users/"+scimID,
			patchOp("replace", "active", false))
		if patchRec.Code != http.StatusOK {
			t.Fatalf("setup: disable: expected 200, got %d body=%v", patchRec.Code, patched)
		}

		readRec, read := doScimGet(t, e, tokenStr, "/scim/v2/Users/"+scimID)
		if readRec.Code != http.StatusOK {
			t.Fatalf("a disabled user must stay readable, got %d body=%v", readRec.Code, read)
		}
		if read["active"] != false {
			t.Errorf("active = %v, want false", read["active"])
		}
		_, list := doScimGet(t, e, tokenStr, "/scim/v2/Users")
		if ids := scimResourceIDs(t, list); len(ids) != 1 || ids[0] != scimID {
			t.Errorf("Resources ids = %v, want the disabled user to remain listed", ids)
		}
	})

	t.Run("recreating the same userName gives 409 uniqueness", func(t *testing.T) {
		// 採用していない側。§3.6 は削除済みを一意性判定から外すことを SHOULD として
		// 求めるが、preferred_username の部分一意索引は PendingDeletion の行を残す。
		h := newScimHarness()
		e := h.echo
		tokenStr := issueAllScimToken(t, h.apiTokens)

		scimID := createScimUserForTest(t, e, tokenStr, map[string]any{"userName": "recycled@example.com"})
		deleteScimUserForTest(t, e, tokenStr, scimID)

		rec, body := doScimJSON(t, e, http.MethodPost, tokenStr, "/scim/v2/Users", map[string]any{
			"userName": "recycled@example.com",
		})
		if rec.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d body=%v", rec.Code, body)
		}
		if body["scimType"] != "uniqueness" {
			t.Errorf("scimType = %v, want uniqueness", body["scimType"])
		}
	})
}
