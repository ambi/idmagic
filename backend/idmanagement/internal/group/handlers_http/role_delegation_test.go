package handlers_http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-IDMANAGEMENT-092, REQ-IDMANAGEMENT-093: 管理 HTTP で承認、取消しと不正な委任の無作用を観測する。
//verify:operation SetGroupRoleDelegation: 現行の式と業務ロールを承認した JSON、取消しの保存、入力不正と他テナントの問題応答。
func TestGroupRoleDelegationHTTPContract(t *testing.T) {
	e, repo := newAdminGroupHandler(t)
	csrf, cookie := adminCSRF(t, e)
	ctx := context.Background()
	now := time.Now().UTC()
	g := &groupdomain.Group{ID: "automatic", TenantID: tenancydomain.DefaultTenantID, Name: "automatic", MembershipType: groupdomain.GroupMembershipDynamic, Roles: []string{"catalog:read"}, CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(ctx, g); err != nil {
		t.Fatal(err)
	}
	expression := `user.preferred_username == "alice"`
	if err := repo.SaveDynamicRule(ctx, &groupdomain.DynamicGroupRule{TenantID: g.TenantID, GroupID: g.ID, Expression: expression, Enabled: true, Version: 1, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AddMember(ctx, &groupdomain.GroupMember{GroupID: g.ID, UserID: "alice", Source: groupdomain.MembershipSourceDynamicRule, RuleVersion: new(int64(1)), CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	path := "/api/admin/v1/groups/automatic/role-delegation"
	assertRoles := func(want []string) {
		t.Helper()
		groups, err := repo.ListGroupsByUser(ctx, g.TenantID, "alice")
		if err != nil || len(groups) != 1 || !slices.Equal(groups[0].Roles, want) {
			t.Fatalf("groups=%+v err=%v want=%v", groups, err, want)
		}
	}
	assertRoles(nil)
	for _, body := range []any{
		map[string]any{"roles": []string{"catalog:write"}, "expression": expression},
		map[string]any{"roles": g.Roles, "expression": "true"},
		map[string]any{"roles": []string{"admin"}, "expression": expression},
	} {
		res := adminJSONRequest(t, e, http.MethodPut, path, csrf, cookie, body)
		if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "invalid_role") {
			t.Fatalf("status=%d body=%s", res.Code, res.Body)
		}
		assertRoles(nil)
	}
	for _, body := range []any{map[string]any{}, map[string]any{"roles": []string{}}, map[string]any{"roles": nil, "expression": ""}} {
		res := adminJSONRequest(t, e, http.MethodPut, path, csrf, cookie, body)
		if res.Code != http.StatusBadRequest {
			t.Fatalf("missing input status=%d body=%s", res.Code, res.Body)
		}
	}
	res := adminJSONRequest(t, e, http.MethodPut, path, csrf, cookie, map[string]any{"roles": g.Roles, "expression": expression})
	if res.Code != http.StatusOK || !strings.Contains(res.Header().Get("Cache-Control"), "no-store") {
		t.Fatalf("approval status=%d body=%s", res.Code, res.Body)
	}
	var approved struct {
		RoleDelegation *groupdomain.GroupRoleDelegation `json:"role_delegation"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &approved); err != nil {
		t.Fatal(err)
	}
	if approved.RoleDelegation == nil || approved.RoleDelegation.ApprovedBy != "admin" || approved.RoleDelegation.Expression != expression || !slices.Equal(approved.RoleDelegation.Roles, g.Roles) || approved.RoleDelegation.ApprovedAt.IsZero() {
		t.Fatalf("approval=%+v", approved.RoleDelegation)
	}
	assertRoles(g.Roles)
	res = adminJSONRequest(t, e, http.MethodPut, path, csrf, cookie, map[string]any{"roles": []string{}, "expression": ""})
	if res.Code != http.StatusOK || strings.Contains(res.Body.String(), "role_delegation") {
		t.Fatalf("revocation status=%d body=%s", res.Code, res.Body)
	}
	assertRoles(nil)
	foreign := *g
	foreign.ID, foreign.TenantID = "foreign", "another-tenant"
	if err := repo.Save(ctx, &foreign); err != nil {
		t.Fatal(err)
	}
	res = adminJSONRequest(t, e, http.MethodPut, "/api/admin/v1/groups/foreign/role-delegation", csrf, cookie, map[string]any{"roles": g.Roles, "expression": expression})
	if res.Code != http.StatusNotFound || !strings.Contains(res.Body.String(), "group_not_found") {
		t.Fatalf("foreign status=%d body=%s", res.Code, res.Body)
	}
	// 業務ロールへの委任があっても、一般利用者自身は承認できない。
	request := httptest.NewRequest(http.MethodPut, defaultRealmPath(path), strings.NewReader(`{"roles":[],"expression":""}`))
	request.Header.Set("X-Demo-Sub", "alice")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", csrf)
	request.AddCookie(cookie)
	denied := httptest.NewRecorder()
	e.ServeHTTP(denied, request)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("ordinary actor status=%d body=%s", denied.Code, denied.Body)
	}
	assertRoles(nil)
}

//spec:covers REQ-IDMANAGEMENT-092: 自動所属の管理ロール設定を入口で拒否し、グループを保存しない。
func TestAutomaticGroupCannotConfigureAdministrativeRoles(t *testing.T) {
	e, _ := newAdminGroupHandler(t)
	csrf, cookie := adminCSRF(t, e)
	before := adminGroupCount(t, e)
	for _, role := range []string{"admin", "system_admin"} {
		res := adminJSONRequest(t, e, http.MethodPost, "/api/admin/v1/groups", csrf, cookie, map[string]any{"name": role, "membership_type": "dynamic", "roles": []string{role}, "dynamic_rule": map[string]any{"expression": `user.email_verified == true`}})
		if res.Code != http.StatusUnprocessableEntity || !strings.Contains(res.Body.String(), "invalid_role") {
			t.Fatalf("create status=%d body=%s", res.Code, res.Body)
		}
	}
	if got := adminGroupCount(t, e); got != before {
		t.Fatalf("group count changed: %d -> %d", before, got)
	}
}
