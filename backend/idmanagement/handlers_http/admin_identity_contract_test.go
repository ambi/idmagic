package handlers_http_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func idmContractAdminUser(t *testing.T, body map[string]any, id, username, status string) {
	t.Helper()
	if body["id"] != id || body["preferred_username"] != username || body["status"] != status {
		t.Fatalf("user=%v", body)
	}
	for _, key := range []string{"email_verified", "mfa_enrolled"} {
		if _, ok := body[key].(bool); !ok {
			t.Fatalf("%s=%v", key, body[key])
		}
	}
	roles, present := body["roles"]
	if _, ok := roles.([]any); !present || (!ok && roles != nil) {
		t.Fatalf("roles=%v present=%v", roles, present)
	}
	for _, key := range []string{"created_at", "updated_at"} {
		idmContractTime(t, body, key)
	}
	if _, present := body["password_hash"]; present {
		t.Fatalf("user exposes password hash: %v", body)
	}
}

//verify:operation CreateAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation GetAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation UpdateAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation DisableAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation EnableAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation DeleteAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation RestoreAdminUser: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation SetUserRequiredAction: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
//verify:operation ClearUserRequiredAction: ユーザーの作成、取得、更新、状態変更、required action の応答と保存結果
func TestAdminUserHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	admin := f.seedSession(t, "user-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin, withFreshStepUp)
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		return f.send(t, idmRefusalRequest{method: method, path: path, body: body, sessionID: admin, csrf: idmRefusalCSRF})
	}
	const base = "/api/admin/v1/users"
	created := idmContractJSON(t, send(http.MethodPost, base, map[string]any{
		"preferred_username": "contract-user", "password": "Password-Contract-123!", "name": "Contract User", "email": "contract-user@example.test", "roles": []string{"support"},
	}), http.StatusCreated)
	id, ok := created["id"].(string)
	if !ok || id == "" {
		t.Fatalf("created user=%v", created)
	}
	idmContractAdminUser(t, created, id, "contract-user", "active")
	if created["name"] != "Contract User" || created["email"] != "contract-user@example.test" || created["email_verified"] != false || created["mfa_enrolled"] != false || !reflect.DeepEqual(created["roles"], []any{"support"}) {
		t.Fatalf("created user=%v", created)
	}
	seeded := idmContractJSON(t, send(http.MethodGet, base+"/"+idmRefusalAlice, nil), http.StatusOK)
	idmContractAdminUser(t, seeded, idmRefusalAlice, idmRefusalAlice, "active")
	if _, present := seeded["tenant_id"]; present || seeded["roles"] != nil {
		t.Fatalf("seeded user's HTTP projection=%v", seeded)
	}
	path := base + "/" + id
	get := func() map[string]any {
		return idmContractJSON(t, send(http.MethodGet, path, nil), http.StatusOK)
	}
	if stored := get(); !reflect.DeepEqual(stored, created) {
		t.Fatalf("stored user=%v created=%v", stored, created)
	}
	updated := idmContractJSON(t, send(http.MethodPatch, path, map[string]any{"preferred_username": "contract-renamed", "name": "Renamed", "email": "renamed@example.test", "roles": []string{"helpdesk"}}), http.StatusOK)
	idmContractAdminUser(t, updated, id, "contract-renamed", "active")
	if updated["name"] != "Renamed" || updated["email"] != "renamed@example.test" || updated["created_at"] != created["created_at"] || !reflect.DeepEqual(updated["roles"], []any{"helpdesk"}) || !reflect.DeepEqual(updated, get()) {
		t.Fatalf("updated user=%v", updated)
	}
	set := idmContractJSON(t, send(http.MethodPost, path+"/required-actions", map[string]any{"action": "verify_email"}), http.StatusOK)
	idmContractAdminUser(t, set, id, "contract-renamed", "active")
	if !reflect.DeepEqual(set["required_actions"], []any{"verify_email"}) || !reflect.DeepEqual(set, get()) {
		t.Fatalf("required actions=%v", set)
	}
	cleared := idmContractJSON(t, send(http.MethodDelete, path+"/required-actions/verify_email", nil), http.StatusOK)
	idmContractAdminUser(t, cleared, id, "contract-renamed", "active")
	if _, present := cleared["required_actions"]; present || !reflect.DeepEqual(cleared, get()) {
		t.Fatalf("cleared user=%v", cleared)
	}
	idmContractEmpty(t, send(http.MethodPost, path+"/disable", nil))
	disabled := get()
	idmContractAdminUser(t, disabled, id, "contract-renamed", "disabled")
	idmContractTime(t, disabled, "disabled_at")
	idmContractEmpty(t, send(http.MethodPost, path+"/enable", nil))
	idmContractAdminUser(t, get(), id, "contract-renamed", "active")
	idmContractEmpty(t, send(http.MethodDelete, path, map[string]any{"reason": "contract"}))
	pending := get()
	idmContractAdminUser(t, pending, id, "contract-renamed", "pending_deletion")
	idmContractTime(t, pending, "pending_deletion_at")
	idmContractTime(t, pending, "purge_after")
	restored := idmContractJSON(t, send(http.MethodPost, path+"/restore", nil), http.StatusOK)
	idmContractAdminUser(t, restored, id, "contract-renamed", "active")
	if !reflect.DeepEqual(restored, get()) {
		t.Fatalf("restored user=%v", restored)
	}
	idmContractProblem(t, send(http.MethodPost, path+"/restore", nil), http.StatusConflict, "not_pending_deletion")
	idmContractProblem(t, send(http.MethodPost, base, map[string]any{"preferred_username": "contract-renamed", "password": "Password-Contract-123!"}), http.StatusConflict, "username_conflict")
}

func idmContractGroup(t *testing.T, body map[string]any, name, membershipType string, count int) string {
	t.Helper()
	id, ok := body["id"].(string)
	if !ok || id == "" || body["tenant_id"] != tenancydomain.DefaultTenantID || body["name"] != name || body["membership_type"] != membershipType || body["member_count"] != float64(count) {
		t.Fatalf("group=%v", body)
	}
	roles, present := body["roles"]
	if _, ok := roles.([]any); !present || (!ok && roles != nil) {
		t.Fatalf("group roles=%v", body)
	}
	idmContractTime(t, body, "created_at")
	return id
}

//verify:operation CreateGroup: グループと member の HTTP 応答および保存した所属
//verify:operation GetGroup: グループと member の HTTP 応答および保存した所属
//verify:operation UpdateGroup: グループと member の HTTP 応答および保存した所属
//verify:operation DeleteGroup: グループと member の HTTP 応答および保存した所属
//verify:operation AddGroupMember: グループと member の HTTP 応答および保存した所属
//verify:operation RemoveGroupMember: グループと member の HTTP 応答および保存した所属
//verify:operation ListUserGroups: グループと member の HTTP 応答および保存した所属
func TestManualGroupHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	admin := f.seedSession(t, "group-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	send := func(method, path string, body any) *httptest.ResponseRecorder {
		return f.send(t, idmRefusalRequest{method: method, path: path, body: body, sessionID: admin, csrf: idmRefusalCSRF})
	}
	const base = "/api/admin/v1/groups"
	created := idmContractJSON(t, send(http.MethodPost, base, map[string]any{"name": "contract-group", "description": "Contract", "email": "group@example.test", "roles": []string{"support"}}), http.StatusCreated)
	id := idmContractGroup(t, created, "contract-group", "manual", 0)
	if created["description"] != "Contract" || created["email"] != "group@example.test" || !reflect.DeepEqual(created["roles"], []any{"support"}) {
		t.Fatalf("created group=%v", created)
	}
	path := base + "/" + id
	detail := func() map[string]any { return idmContractJSON(t, send(http.MethodGet, path, nil), http.StatusOK) }
	if got := detail(); !reflect.DeepEqual(got["group"], created) || !reflect.DeepEqual(got["members"], []any{}) {
		t.Fatalf("stored group=%v", got)
	}
	updated := idmContractJSON(t, send(http.MethodPatch, path, map[string]any{"name": "contract-updated", "roles": []string{"helpdesk"}}), http.StatusOK)
	if idmContractGroup(t, updated, "contract-updated", "manual", 0) != id || updated["created_at"] != created["created_at"] || !reflect.DeepEqual(updated["roles"], []any{"helpdesk"}) || !reflect.DeepEqual(detail()["group"], updated) {
		t.Fatalf("updated group=%v", updated)
	}
	idmContractTime(t, updated, "updated_at")
	memberPath := path + "/members/" + idmRefusalBob
	idmContractEmpty(t, send(http.MethodPost, memberPath, nil))
	members := detail()
	idmContractGroup(t, members["group"].(map[string]any), "contract-updated", "manual", 1)
	list, ok := members["members"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("members=%v", members)
	}
	member := list[0].(map[string]any)
	if member["user_id"] != idmRefusalBob || member["preferred_username"] != idmRefusalBob {
		t.Fatalf("member=%v", member)
	}
	idmContractTime(t, member, "created_at")
	userGroups := idmContractJSON(t, send(http.MethodGet, "/api/admin/v1/users/"+idmRefusalBob+"/groups", nil), http.StatusOK)
	groups, ok := userGroups["groups"].([]any)
	if !ok || len(groups) != 1 || groups[0].(map[string]any)["id"] != id || !reflect.DeepEqual(userGroups["direct_roles"], []any{}) || !reflect.DeepEqual(userGroups["group_roles"], []any{"helpdesk"}) || !reflect.DeepEqual(userGroups["effective_roles"], []any{"helpdesk"}) {
		t.Fatalf("user groups=%v", userGroups)
	}
	idmContractEmpty(t, send(http.MethodDelete, memberPath, nil))
	after := detail()
	idmContractGroup(t, after["group"].(map[string]any), "contract-updated", "manual", 0)
	if !reflect.DeepEqual(after["members"], []any{}) {
		t.Fatalf("remaining members=%v", after)
	}
	idmContractEmpty(t, send(http.MethodDelete, path, nil))
	listed := idmContractJSON(t, send(http.MethodGet, base, nil), http.StatusOK)
	for _, raw := range listed["groups"].([]any) {
		if raw.(map[string]any)["id"] == id {
			t.Fatalf("deleted group remains: %v", listed)
		}
	}
}

func idmContractDynamicRule(t *testing.T, rule map[string]any, expression string, enabled bool, version int) {
	t.Helper()
	if rule["group_id"] != idmRefusalDynamicGroup || rule["tenant_id"] != tenancydomain.DefaultTenantID || rule["expression"] != expression || rule["enabled"] != enabled || rule["version"] != float64(version) || !reflect.DeepEqual(rule["referenced_attributes"], []any{"preferred_username"}) {
		t.Fatalf("dynamic rule=%v", rule)
	}
	for _, key := range []string{"created_at", "updated_at"} {
		idmContractTime(t, rule, key)
	}
}

//verify:operation UpdateDynamicGroupRule: 動的ルールの更新、preview、無効化、有効化の HTTP 応答と保存結果
//verify:operation PreviewDynamicGroupRule: 動的ルールの更新、preview、無効化、有効化の HTTP 応答と保存結果
//verify:operation EnableDynamicGroupRule: 動的ルールの更新、preview、無効化、有効化の HTTP 応答と保存結果
//verify:operation DisableDynamicGroupRule: 動的ルールの更新、preview、無効化、有効化の HTTP 応答と保存結果
func TestDynamicGroupHTTPContracts(t *testing.T) {
	f := newIdmRefusalServer(t)
	admin := f.seedSession(t, "dynamic-contract", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	detail := idmContractJSON(t, f.send(t, idmRefusalRequest{method: http.MethodGet, path: "/api/admin/v1/groups/" + idmRefusalDynamicGroup, sessionID: admin}), http.StatusOK)
	group := detail["group"].(map[string]any)
	idmContractGroup(t, group, "dynamic-engineering", "dynamic", 0)
	if group["roles"] != nil {
		t.Fatalf("empty group roles=%v", group)
	}
	path := "/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/dynamic-rule"
	send := func(method, suffix string, body any) *httptest.ResponseRecorder {
		return f.send(t, idmRefusalRequest{method: method, path: path + suffix, body: body, sessionID: admin, csrf: idmRefusalCSRF})
	}
	expression := `user.preferred_username == "` + idmRefusalAlice + `"`
	rule := idmContractJSON(t, send(http.MethodPut, "", map[string]any{"expression": expression}), http.StatusOK)
	idmContractDynamicRule(t, rule, expression, false, 1)
	preview := idmContractJSON(t, send(http.MethodPost, "/preview", map[string]any{"expression": expression, "user_ids": []string{idmRefusalAlice, idmRefusalBob}}), http.StatusOK)
	results, ok := preview["results"].([]any)
	if !ok || len(results) != 2 {
		t.Fatalf("preview=%v", preview)
	}
	for index, userID := range []string{idmRefusalAlice, idmRefusalBob} {
		result := results[index].(map[string]any)
		change := "unchanged"
		if index == 0 {
			change = "add"
		}
		if result["user_id"] != userID || result["matched"] != (index == 0) || result["change"] != change {
			t.Fatalf("result=%v", result)
		}
	}
	enabled := idmContractJSON(t, send(http.MethodPost, "/enable", nil), http.StatusOK)
	idmContractDynamicRule(t, enabled, expression, true, 2)
	disabled := idmContractJSON(t, send(http.MethodPost, "/disable", nil), http.StatusOK)
	idmContractDynamicRule(t, disabled, expression, false, 3)
	idmContractProblem(t, send(http.MethodPut, "", map[string]any{"expression": "user.roles == []"}), http.StatusUnprocessableEntity, "invalid_dynamic_group_rule")
}

//verify:operation CreateAdminUser: 非管理者への 403 Problem Details
//verify:operation GetAdminUser: 非管理者への 403 Problem Details
//verify:operation UpdateAdminUser: 非管理者への 403 Problem Details
//verify:operation DisableAdminUser: 非管理者への 403 Problem Details
//verify:operation EnableAdminUser: 非管理者への 403 Problem Details
//verify:operation DeleteAdminUser: 非管理者への 403 Problem Details
//verify:operation RestoreAdminUser: 非管理者への 403 Problem Details
//verify:operation SetUserRequiredAction: 非管理者への 403 Problem Details
//verify:operation ClearUserRequiredAction: 非管理者への 403 Problem Details
//verify:operation CreateGroup: 非管理者への 403 Problem Details
//verify:operation GetGroup: 非管理者への 403 Problem Details
//verify:operation UpdateGroup: 非管理者への 403 Problem Details
//verify:operation DeleteGroup: 非管理者への 403 Problem Details
//verify:operation AddGroupMember: 非管理者への 403 Problem Details
//verify:operation RemoveGroupMember: 非管理者への 403 Problem Details
//verify:operation ListUserGroups: 非管理者への 403 Problem Details
//verify:operation UpdateDynamicGroupRule: 非管理者への 403 Problem Details
//verify:operation PreviewDynamicGroupRule: 非管理者への 403 Problem Details
//verify:operation EnableDynamicGroupRule: 非管理者への 403 Problem Details
//verify:operation DisableDynamicGroupRule: 非管理者への 403 Problem Details
func TestAdminIdentityHTTPContractsRefuseNonAdmin(t *testing.T) {
	f := newIdmRefusalServer(t)
	user := f.seedSession(t, "identity-contract-refused", tenancydomain.DefaultTenantID, idmRefusalBob, withFreshStepUp)
	userPath := "/api/admin/v1/users/" + idmRefusalAlice
	groupPath := "/api/admin/v1/groups/" + idmRefusalManualGroup
	dynamicPath := "/api/admin/v1/groups/" + idmRefusalDynamicGroup + "/dynamic-rule"
	for _, test := range []struct{ method, path string }{
		{http.MethodPost, "/api/admin/v1/users"},
		{http.MethodGet, userPath},
		{http.MethodPatch, userPath},
		{http.MethodPost, userPath + "/disable"},
		{http.MethodPost, userPath + "/enable"},
		{http.MethodDelete, userPath},
		{http.MethodPost, userPath + "/restore"},
		{http.MethodPost, userPath + "/required-actions"},
		{http.MethodDelete, userPath + "/required-actions/verify_email"},
		{http.MethodPost, "/api/admin/v1/groups"},
		{http.MethodGet, groupPath},
		{http.MethodPatch, groupPath},
		{http.MethodDelete, groupPath},
		{http.MethodPost, groupPath + "/members/" + idmRefusalBob},
		{http.MethodDelete, groupPath + "/members/" + idmRefusalAlice},
		{http.MethodGet, userPath + "/groups"},
		{http.MethodPut, dynamicPath},
		{http.MethodPost, dynamicPath + "/preview"},
		{http.MethodPost, dynamicPath + "/enable"},
		{http.MethodPost, dynamicPath + "/disable"},
	} {
		t.Run(test.method+test.path, func(t *testing.T) {
			idmContractProblem(t, f.send(t, idmRefusalRequest{method: test.method, path: test.path, sessionID: user, csrf: idmRefusalCSRF, body: map[string]any{}}), http.StatusForbidden, "access_denied")
		})
	}
	if len(*f.events) != 0 || len(f.runnableJobs(t)) != 0 {
		t.Fatal("refused requests emitted changes")
	}
}
