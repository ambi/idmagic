package handlers_http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	oauth2http "github.com/ambi/idmagic/backend/oauth2/handlers_http"
)

func TestAdminRolePoliciesOmitInternalDocReferences(t *testing.T) {
	e, _, _ := newKeyAdminServer(t, keyAdminUser("admin", "acme", []string{"admin"}))
	rec := getAdminRolePolicies(e, "/realms/acme/api/admin/v1/policy/roles")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, leak := range []string{"actor.roles", "allow_when", `"requirements"`} {
		if strings.Contains(body, leak) {
			t.Fatalf("response leaks internal token %q: %s", leak, body)
		}
	}
	// 説明文 (description) には設計者向けの内部語を出さない。interface 名/path は
	// 技術的識別子なので対象外とし、description のみを検査する。
	internalTerms := []string{
		"User.roles", "SystemAdministrator", "tombstone", "reject", "Consent", "SCL",
	}
	for _, role := range decodeAdminRolePolicies(t, rec) {
		descriptions := []string{role.Description}
		for _, permission := range role.Permissions {
			descriptions = append(descriptions, permission.Description)
		}
		for _, description := range descriptions {
			for _, term := range internalTerms {
				if strings.Contains(description, term) {
					t.Fatalf("description leaks internal term %q: %q", term, description)
				}
			}
		}
	}
}

// EX-OAUTH2-004-02: admin でも system_admin でもない主体のロールポリシー一覧は
// 拒否され、レスポンスボディにロールポリシーが 1 件も含まれない。
//
// 403 を書いてから一覧も書く実装はステータスだけを読むテストを通すので、
// 本文にロールポリシーが漏れていないことまで読み直す。
func TestAdminRolePoliciesRequireAdminRole(t *testing.T) {
	e, _, _ := newKeyAdminServer(t, keyAdminUser("plain", "acme", nil))
	rec := getAdminRolePolicies(e, "/realms/acme/api/admin/v1/policy/roles")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, leak := range []string{`"roles"`, `"permissions"`, "system_admin"} {
		if strings.Contains(rec.Body.String(), leak) {
			t.Fatalf("拒否された応答がロールポリシー %q を含む: %s", leak, rec.Body.String())
		}
	}
}

func TestAdminRolePoliciesHideSystemAdminPermissionsFromAdmin(t *testing.T) {
	e, _, _ := newKeyAdminServer(t, keyAdminUser("admin", "acme", []string{"admin"}))
	rec := getAdminRolePolicies(e, "/realms/acme/api/admin/v1/policy/roles")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	roles := decodeAdminRolePolicies(t, rec)
	if hasAdminRolePermission(roles, "system_admin", "AdminTenantsManage") {
		t.Fatal("AdminTenantsManage is visible to plain admin")
	}
	if !hasAdminRolePermission(roles, "admin", "AdminUserRead") {
		t.Fatal("AdminUserRead is missing")
	}
}

func TestAdminRolePoliciesIncludeControlPlanePermissionsForSystemAdmin(t *testing.T) {
	e, _, _ := newKeyAdminServer(
		t,
		keyAdminUser("ops", tenancydomain.DefaultTenantID, []string{"system_admin"}),
	)
	rec := getAdminRolePolicies(e, "/realms/default/api/admin/v1/policy/roles")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	roles := decodeAdminRolePolicies(t, rec)
	for _, permission := range []string{"AdminTenantsManage", "SystemKeyHealthRead"} {
		if !hasAdminRolePermission(roles, "system_admin", permission) {
			t.Fatalf("%s is missing", permission)
		}
	}
}

func getAdminRolePolicies(e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, defaultRealmPath(path), http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeAdminRolePolicies(t *testing.T, rec *httptest.ResponseRecorder) []oauth2http.AdminRolePolicyResponse {
	t.Helper()
	var body struct {
		Roles []oauth2http.AdminRolePolicyResponse `json:"roles"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Roles
}

func hasAdminRolePermission(roles []oauth2http.AdminRolePolicyResponse, roleName, permissionName string) bool {
	for _, role := range roles {
		if role.Name != roleName {
			continue
		}
		for _, permission := range role.Permissions {
			if permission.Name == permissionName {
				return true
			}
		}
	}
	return false
}

// EX-OAUTH2-004-01: 認証済みの管理者が受け取るロールポリシー一覧には、参照可能なロールと、
// その権限と、権限ごとの HTTP インターフェース (名前、メソッド、パス) が入っている。
//
// 3 つを別々に読むのは、どれか 1 つを落としても他の 2 つは揃う実装があるためである。
// とくに interfaces は入れ子の最下層なので、ロールと権限だけを読むテストは、
// interfaces を常に空配列で返す実装を通してしまう。実際の対応まで読むために、
// `AdminUserRead` が GET /api/admin/v1/users を指していることを名指しで確かめる。
func TestAdminRolePoliciesListVisibleRolesPermissionsAndInterfaces(t *testing.T) {
	e, _, _ := newKeyAdminServer(t, keyAdminUser("admin", "acme", []string{"admin"}))
	rec := getAdminRolePolicies(e, "/realms/acme/api/admin/v1/policy/roles")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	roles := decodeAdminRolePolicies(t, rec)

	// 参照可能なロールが含まれる。
	var admin *oauth2http.AdminRolePolicyResponse
	for i, role := range roles {
		if role.Name == "admin" {
			admin = &roles[i]
		}
	}
	if admin == nil {
		t.Fatalf("参照可能なロール admin が一覧に無い: %+v", roles)
	}

	// その権限が含まれる。
	found := false
	// 対応する HTTP インターフェースが含まれる。
	interfaced := false
	for _, permission := range admin.Permissions {
		if permission.Name != "AdminUserRead" {
			continue
		}
		found = true
		for _, iface := range permission.Interfaces {
			if iface.Name == "ListAdminUsers" && iface.Method == http.MethodGet &&
				iface.Path == "/api/admin/v1/users" {
				interfaced = true
			}
		}
		if !interfaced {
			t.Fatalf("AdminUserRead に GET /api/admin/v1/users の対応が無い: %+v", permission.Interfaces)
		}
	}
	if !found {
		t.Fatalf("admin の権限 AdminUserRead が一覧に無い: %+v", admin.Permissions)
	}
}
