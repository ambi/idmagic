package handlers_http_test

// Application 管理 API が宣言する 404 を、HTTP の境界で固定する。
//
// 404 を書くのは `writeApplicationError` と `writeCategoryError` で、`mise run check-status-drift`
// はこれらのヘルパーの中まで追跡しない。そこで各 API 操作へ別テナントの id と存在しない id を
// 送り、宣言どおりの 404 であることと、両者の応答を区別できないことを確かめる。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

const missingID = "ffffffff-ffff-4fff-bfff-ffffffffffff"

//spec:covers REQ-APPLICATION-007: Application を指定する管理 API 16 件が、別テナントの Application を存在しない id と同じ 404 application_not_found で拒否し、その Application を変えないこと。
func TestApplicationAdminOperationsAnswerAnotherTenantsApplicationAsMissing(t *testing.T) {
	e := newTwoTenantApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createSecretApplication(t, e, csrf, cookie)
	credentials := readSecretCredentials(t, e, csrf, cookie, applicationID)
	if len(credentials) != 1 {
		t.Fatalf("前提が壊れている: 資格情報が %d 件", len(credentials))
	}
	credentialID := credentials[0].CredentialID
	// 保存されていないサインイン方針は読むたびに現在時刻で合成されるので、先に保存して前後の比較を安定させる。
	for _, path := range []string{"/api/admin/v1/applications/" + applicationID + "/sign-in-policy", "/api/admin/v1/default-sign-in-policy"} {
		if saved := adminJSON(t, e, http.MethodPut, path, csrf, cookie, map[string]any{"rules": []any{}}); saved.Code != http.StatusOK {
			t.Fatalf("save %s status=%d body=%s", path, saved.Code, saved.Body.String())
		}
	}
	before := ownTenantState(t, e, csrf, cookie, applicationID)

	cases := []struct {
		operation, method, suffix string
		body                      any
	}{
		{"UpdateAdminApplication", http.MethodPatch, "", map[string]any{"name": "taken over"}},
		{"DeleteAdminApplication", http.MethodDelete, "", nil},
		{"DeleteApplicationIcon", http.MethodDelete, "/icon", nil},
		{"UpdateApplicationOidcConfig", http.MethodPatch, "/oidc", map[string]any{"redirect_uris": []string{"https://attacker.example/cb"}}},
		{"RotateApplicationClientSecret", http.MethodPost, "/oidc/rotate-secret", map[string]any{"grace_days": 7}},
		{"IssueApplicationClientSecret", http.MethodPost, "/oidc/client-secrets", map[string]any{"expires_in_days": 90}},
		{"RevokeApplicationClientSecret", http.MethodDelete, "/oidc/client-secrets/" + credentialID, nil},
		{"UpdateApplicationWsFedConfig", http.MethodPatch, "/wsfed", map[string]any{"reply_url": "https://attacker.example/wsfed"}},
		{"UpdateApplicationSamlConfig", http.MethodPatch, "/saml", map[string]any{"acs_url": "https://attacker.example/acs"}},
		{"ListApplicationAssignments", http.MethodGet, "/assignments", nil},
		{"AssignApplication", http.MethodPost, "/assignments", map[string]any{"subject_type": "user", "subject_id": "globex-admin"}},
		{"GetAppSignInPolicy", http.MethodGet, "/sign-in-policy", nil},
		{"UpdateAppSignInPolicy", http.MethodPut, "/sign-in-policy", map[string]any{"rules": []any{}}},
		{"SetApplicationCategories", http.MethodPut, "/categories", map[string]any{"category_ids": []string{}}},
	}
	globex := newRealmClient(t, e, "globex", "globex-admin")
	for _, c := range cases {
		t.Run(c.operation, func(t *testing.T) {
			foreign := globex.json(t, c.method, "/api/admin/v1/applications/"+applicationID+c.suffix, c.body)
			missing := globex.json(t, c.method, "/api/admin/v1/applications/"+missingID+c.suffix, c.body)
			assertNotFoundProblem(t, foreign, "urn:idmagic:error:application_not_found")
			assertIndistinguishableFromMissing(t, foreign, missing)
		})
	}
	t.Run("UploadApplicationIcon", func(t *testing.T) {
		foreign := globex.multipart(t, "/api/admin/v1/applications/"+applicationID+"/icon", gifBytes)
		missing := globex.multipart(t, "/api/admin/v1/applications/"+missingID+"/icon", gifBytes)
		assertNotFoundProblem(t, foreign, "urn:idmagic:error:application_not_found")
		assertIndistinguishableFromMissing(t, foreign, missing)
	})

	// 拒否が防いだ効果。default テナントの Application は、設定も資格情報も割り当ても変わらない。
	if after := ownTenantState(t, e, csrf, cookie, applicationID); after != before {
		t.Fatalf("別テナントからの要求が Application を変えた:\nbefore=%s\nafter=%s", before, after)
	}
}

//spec:covers REQ-APPLICATION-007: カテゴリーの更新と削除が、別テナントのカテゴリーを存在しない id と同じ 404 category_not_found で拒否し、そのカテゴリーを変えないこと。
func TestApplicationCategoryOperationsAnswerAnotherTenantsCategoryAsMissing(t *testing.T) {
	e := newTwoTenantApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	categoryID := createCategory(t, e, csrf, cookie, "Finance")
	listCategories := func() string {
		response := adminJSON(t, e, http.MethodGet, "/api/admin/v1/application-categories", csrf, cookie, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("list categories status=%d body=%s", response.Code, response.Body.String())
		}
		return response.Body.String()
	}
	before := listCategories()

	globex := newRealmClient(t, e, "globex", "globex-admin")
	for _, c := range []struct {
		operation, method string
		body              any
	}{
		{"UpdateApplicationCategory", http.MethodPatch, map[string]any{"name": "taken over"}},
		{"DeleteApplicationCategory", http.MethodDelete, nil},
	} {
		t.Run(c.operation, func(t *testing.T) {
			foreign := globex.json(t, c.method, "/api/admin/v1/application-categories/"+categoryID, c.body)
			missing := globex.json(t, c.method, "/api/admin/v1/application-categories/"+missingID, c.body)
			assertNotFoundProblem(t, foreign, "urn:idmagic:error:category_not_found")
			assertIndistinguishableFromMissing(t, foreign, missing)
		})
	}

	if after := listCategories(); after != before {
		t.Fatalf("別テナントからの要求がカテゴリーを変えた:\nbefore=%s\nafter=%s", before, after)
	}
}

// ownTenantState は、default テナントの管理者から見た Application の詳細と割り当てを連結して返す。
// 別テナントからの要求の前後で比べ、どの API 操作も何も変えなかったことを一度に確かめる。
func ownTenantState(t *testing.T, e *echo.Echo, csrf string, cookie *http.Cookie, applicationID string) string {
	t.Helper()
	detail := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	assignments := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID+"/assignments", csrf, cookie, nil)
	if detail.Code != http.StatusOK || assignments.Code != http.StatusOK {
		t.Fatalf("own tenant detail=%d %s assignments=%d %s",
			detail.Code, detail.Body.String(), assignments.Code, assignments.Body.String())
	}
	return detail.Body.String() + "\n" + assignments.Body.String()
}

func assertNotFoundProblem(t *testing.T, response *httptest.ResponseRecorder, wantType string) {
	t.Helper()
	if response.Code != http.StatusNotFound || readProblemType(t, response) != wantType {
		t.Fatalf("status=%d body=%s, want 404 %s", response.Code, response.Body.String(), wantType)
	}
}

// realmClient は、指定 realm の管理者として CSRF を通した要求を送る。
type realmClient struct {
	e      *echo.Echo
	realm  string
	sub    string
	csrf   string
	cookie *http.Cookie
}

func newRealmClient(t *testing.T, e *echo.Echo, realm, sub string) realmClient {
	t.Helper()
	response := getAs(e, sub, "/realms/"+realm+"/api/auth/account")
	if response.Code != http.StatusOK {
		t.Fatalf("account status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	return realmClient{e: e, realm: realm, sub: sub, csrf: body.CSRFToken, cookie: cookies[0]}
}

func (c realmClient) json(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, "/realms/"+c.realm+path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	return c.send(request)
}

func (c realmClient) multipart(t *testing.T, path string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	body, contentType := multipartFile(t, "icon.gif", data)
	request := httptest.NewRequest(http.MethodPost, "/realms/"+c.realm+path, body)
	request.Header.Set("Content-Type", contentType)
	return c.send(request)
}

func (c realmClient) send(request *http.Request) *httptest.ResponseRecorder {
	request.Header.Set("Origin", "http://idp.test")
	request.Header.Set("X-Csrf-Token", c.csrf)
	request.Header.Set("X-Demo-Sub", c.sub)
	request.AddCookie(c.cookie)
	response := httptest.NewRecorder()
	c.e.ServeHTTP(response, request)
	return response
}
