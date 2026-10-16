package handlers_http_test

import (
	"net/http"
	"testing"
)

//spec:covers REQ-APPLICATION-007: 表示設定、カテゴリーと割り当ての変更は HTTP の再読と解除に反映される。
//verify:operation UpdateAdminApplication: 200 JSON は同じ ID と変更した名称、起動 URL、状態を返す。
//verify:operation CreateApplicationCategory: 201 JSON は新しい ID、指定した名称、位置、作成と更新の時刻を返す。
//verify:operation UpdateApplicationCategory: 200 JSON は同じ ID と変更した名称、位置、時刻を返す。
//verify:operation SetApplicationCategories: 200 JSON は保存したカテゴリーの ID と名称を返す。
//verify:operation DeleteApplicationCategory: 削除は空の 204 を返し、再読でカテゴリーがアプリケーションから消える。
//verify:operation UnassignApplication: 解除は空の 204 を返し、一覧で割り当てが消える。
func TestApplicationDisplayAndCategoryHTTPContracts(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	id := createAndAssignWeblink(t, e, csrf, cookie, "Payroll")
	path := "/api/admin/v1/applications/" + id
	updated := applicationContractJSON(t, adminJSON(t, e, http.MethodPatch, path, csrf, cookie, map[string]any{
		"name": "Finance", "status": "disabled", "launch_url": "https://finance.example",
	}), http.StatusOK)
	assertApplicationView(t, updated, id, "Finance", "weblink", "disabled")
	if updated["launch_url"] != "https://finance.example" {
		t.Fatalf("updated=%v", updated)
	}
	created := applicationContractJSON(t, adminJSON(t, e, http.MethodPost, "/api/admin/v1/application-categories", csrf, cookie, map[string]any{
		"name": "Work", "position": 3,
	}), http.StatusCreated)
	category, _ := created["category"].(map[string]any)
	categoryID, _ := category["id"].(string)
	if categoryID == "" || category["name"] != "Work" || category["position"] != float64(3) {
		t.Fatalf("created category=%v", created)
	}
	assertApplicationTime(t, category, "created_at", "updated_at")
	changed := applicationContractJSON(t, adminJSON(t, e, http.MethodPatch, "/api/admin/v1/application-categories/"+categoryID, csrf, cookie, map[string]any{
		"name": "Finance", "position": 1,
	}), http.StatusOK)
	category, _ = changed["category"].(map[string]any)
	if category["id"] != categoryID || category["name"] != "Finance" || category["position"] != float64(1) {
		t.Fatalf("updated category=%v", changed)
	}
	assertApplicationTime(t, category, "created_at", "updated_at")
	set := applicationContractJSON(t, adminJSON(t, e, http.MethodPut, path+"/categories", csrf, cookie, map[string]any{
		"category_ids": []string{categoryID},
	}), http.StatusOK)
	assertApplicationView(t, set, id, "Finance", "weblink", "disabled")
	ids, _ := set["category_ids"].([]any)
	names, _ := set["category_names"].([]any)
	if len(ids) != 1 || ids[0] != categoryID || len(names) != 1 || names[0] != "Finance" {
		t.Fatalf("set categories=%v", set)
	}
	assertApplicationNoContent(t, adminJSON(t, e, http.MethodDelete, "/api/admin/v1/application-categories/"+categoryID, csrf, cookie, nil))
	view := applicationContractJSON(t, adminJSON(t, e, http.MethodGet, path, csrf, cookie, nil), http.StatusOK)
	app, _ := view["application"].(map[string]any)
	assertApplicationView(t, app, id, "Finance", "weblink", "disabled")
	if ids, ok := app["category_ids"].([]any); !ok || len(ids) != 0 {
		t.Fatalf("deleted category retained: %v", app)
	}
	assertApplicationNoContent(t, adminJSON(t, e, http.MethodDelete, path+"/assignments/user/admin", csrf, cookie, nil))
	assignments := applicationContractJSON(t, adminJSON(t, e, http.MethodGet, path+"/assignments", csrf, cookie, nil), http.StatusOK)
	if items, ok := assignments["assignments"].([]any); !ok || len(items) != 0 {
		t.Fatalf("unassigned item retained: %v", assignments)
	}
}

//spec:covers REQ-APPLICATION-013: 非管理者の管理 API は全て 403 access_denied で拒否する。
//verify:operation CreateAdminApplication: 非管理者の作成は 403 の Problem Details を返す。
//verify:operation GetAdminApplication: 非管理者の参照は 403 の Problem Details を返す。
//verify:operation UpdateAdminApplication: 非管理者の更新は 403 の Problem Details を返す。
//verify:operation DeleteAdminApplication: 非管理者の削除は 403 の Problem Details を返す。
//verify:operation UploadApplicationIcon: 非管理者の画像登録は 403 の Problem Details を返す。
//verify:operation DeleteApplicationIcon: 非管理者の画像削除は 403 の Problem Details を返す。
//verify:operation UpdateApplicationOidcConfig: 非管理者の OIDC 更新は 403 の Problem Details を返す。
//verify:operation UpdateApplicationSamlConfig: 非管理者の SAML 更新は 403 の Problem Details を返す。
//verify:operation UpdateApplicationWsFedConfig: 非管理者の WS-Fed 更新は 403 の Problem Details を返す。
//verify:operation IssueApplicationClientSecret: 非管理者の追加発行は 403 の Problem Details を返す。
//verify:operation RevokeApplicationClientSecret: 非管理者の失効は 403 の Problem Details を返す。
//verify:operation RotateApplicationClientSecret: 非管理者のローテーションは 403 の Problem Details を返す。
//verify:operation ListApplicationAssignments: 非管理者の割り当て一覧は 403 の Problem Details を返す。
//verify:operation AssignApplication: 非管理者の割り当ては 403 の Problem Details を返す。
//verify:operation UnassignApplication: 非管理者の割り当て解除は 403 の Problem Details を返す。
//verify:operation GetAppSignInPolicy: 非管理者のポリシー参照は 403 の Problem Details を返す。
//verify:operation UpdateAppSignInPolicy: 非管理者のポリシー更新は 403 の Problem Details を返す。
//verify:operation GetTenantDefaultSignInPolicy: 非管理者の既定ポリシー参照は 403 の Problem Details を返す。
//verify:operation UpdateTenantDefaultSignInPolicy: 非管理者の既定ポリシー更新は 403 の Problem Details を返す。
//verify:operation CreateApplicationCategory: 非管理者のカテゴリー作成は 403 の Problem Details を返す。
//verify:operation UpdateApplicationCategory: 非管理者のカテゴリー更新は 403 の Problem Details を返す。
//verify:operation DeleteApplicationCategory: 非管理者のカテゴリー削除は 403 の Problem Details を返す。
//verify:operation SetApplicationCategories: 非管理者のカテゴリー設定は 403 の Problem Details を返す。
func TestApplicationManagementHTTPContractsRefuseNonAdmin(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	id := createAndAssignWeblink(t, e, csrf, cookie, "Payroll")
	categoryID := createCategory(t, e, csrf, cookie, "Work")
	regular := newRealmClient(t, e, "default", "regular")
	path := "/api/admin/v1/applications/" + id
	category := "/api/admin/v1/application-categories/" + categoryID
	for _, request := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, "/api/admin/v1/applications", map[string]any{"name": "Forbidden", "type": "weblink", "launch_url": "https://forbidden.example"}},
		{http.MethodGet, path, nil},
		{http.MethodPatch, path, map[string]any{"name": "Forbidden"}},
		{http.MethodDelete, path, nil},
		{http.MethodDelete, path + "/icon", nil},
		{http.MethodPatch, path + "/oidc", map[string]any{"scope": "openid"}},
		{http.MethodPatch, path + "/saml", map[string]any{"acs_urls": []string{"https://sp.example/acs"}}},
		{http.MethodPatch, path + "/wsfed", map[string]any{"reply_urls": []string{"https://rp.example/reply"}}},
		{http.MethodPost, path + "/oidc/client-secrets", map[string]any{"expires_in_days": 90}},
		{http.MethodDelete, path + "/oidc/client-secrets/credential", nil},
		{http.MethodPost, path + "/oidc/rotate-secret", map[string]any{"grace_days": 7}},
		{http.MethodGet, path + "/assignments", nil},
		{http.MethodPost, path + "/assignments", map[string]any{"subject_type": "user", "subject_id": "regular"}},
		{http.MethodDelete, path + "/assignments/user/admin", nil},
		{http.MethodGet, path + "/sign-in-policy", nil},
		{http.MethodPut, path + "/sign-in-policy", map[string]any{"rules": []any{}}},
		{http.MethodGet, "/api/admin/v1/default-sign-in-policy", nil},
		{http.MethodPut, "/api/admin/v1/default-sign-in-policy", map[string]any{"rules": []any{}}},
		{http.MethodPost, "/api/admin/v1/application-categories", map[string]any{"name": "Forbidden"}},
		{http.MethodPatch, category, map[string]any{"name": "Forbidden"}},
		{http.MethodDelete, category, nil},
		{http.MethodPut, path + "/categories", map[string]any{"category_ids": []string{categoryID}}},
	} {
		t.Run(request.method+" "+request.path, func(t *testing.T) {
			assertApplicationProblem(t, regular.json(t, request.method, request.path, request.body), http.StatusForbidden, "access_denied")
		})
	}
	assertApplicationProblem(t, regular.multipart(t, path+"/icon", gifBytes), http.StatusForbidden, "access_denied")
	view := applicationContractJSON(t, adminJSON(t, e, http.MethodGet, path, csrf, cookie, nil), http.StatusOK)
	app, _ := view["application"].(map[string]any)
	assertApplicationView(t, app, id, "Payroll", "weblink", "active")
}
