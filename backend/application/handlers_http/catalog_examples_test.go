package handlers_http_test

// Application カタログ (REQ-APPLICATION-007)、アイコン (REQ-APPLICATION-008)、
// 管理 API のロール境界 (REQ-APPLICATION-013) が宣言する具体例を、管理 API の入口で観測する。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pngBytes と gifBytes は、アイコンの種別判定が読む先頭バイト列を持つ最小の画像である。
var (
	pngBytes = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	gifBytes = []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00,")
)

type applicationIconView struct {
	Application struct {
		IconURL       string `json:"icon_url"`
		IconObjectKey string `json:"icon_object_key"`
	} `json:"application"`
}

func readIconFields(t *testing.T, body []byte) applicationIconView {
	t.Helper()
	var view applicationIconView
	if err := json.Unmarshal(body, &view); err != nil {
		t.Fatalf("decode icon fields: %v; body=%s", err, body)
	}
	return view
}

//spec:covers REQ-APPLICATION-007, EX-APPLICATION-007-01: confidential な OIDC アプリケーションの作成応答だけが `client_secret` を一度運び、プロトコル設定の編集と割り当ての保存が読み直せ、取得は同じテナントのものだけを返し、削除までに ApplicationCreated・ApplicationAssigned・ApplicationDeleted が発行されること。
func TestAdminConfiguresAnApplicationAndItsSingleProtocol(t *testing.T) {
	e, events, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	create := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications", csrf, cookie, map[string]any{
		"name": "portal", "type": "oidc", "client_type": "confidential",
		"token_endpoint_auth_method": "client_secret_post",
		"redirect_uris":              []string{"https://portal.example/callback"},
	})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	var created struct {
		Application struct {
			ID string `json:"id"`
		} `json:"application"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v; body=%s", err, create.Body.String())
	}
	if created.ClientSecret == "" {
		t.Fatalf("create response carries no client_secret: %s", create.Body.String())
	}
	applicationID := created.Application.ID

	updated := adminJSON(t, e, http.MethodPatch, "/api/admin/v1/applications/"+applicationID+"/oidc", csrf, cookie, map[string]any{
		"redirect_uris": []string{"https://portal.example/callback/v2"},
		"scope":         "openid profile",
	})
	if updated.Code != http.StatusNoContent {
		t.Fatalf("update oidc status=%d body=%s", updated.Code, updated.Body.String())
	}

	assigned := adminJSON(t, e, http.MethodPost, "/api/admin/v1/applications/"+applicationID+"/assignments", csrf, cookie, map[string]any{
		"subject_type": "user", "subject_id": "alice",
	})
	if assigned.Code != http.StatusCreated {
		t.Fatalf("assign status=%d body=%s", assigned.Code, assigned.Body.String())
	}

	detail := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", detail.Code, detail.Body.String())
	}
	var read struct {
		Oidc struct {
			RedirectURIs []string `json:"redirect_uris"`
			Scope        string   `json:"scope"`
		} `json:"oidc"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &read); err != nil {
		t.Fatalf("decode detail: %v; body=%s", err, detail.Body.String())
	}
	if len(read.Oidc.RedirectURIs) != 1 || read.Oidc.RedirectURIs[0] != "https://portal.example/callback/v2" {
		t.Fatalf("redirect_uris=%v, want the edited one", read.Oidc.RedirectURIs)
	}
	if read.Oidc.Scope != "openid profile" {
		t.Fatalf("scope=%q, want the edited one", read.Oidc.Scope)
	}
	// 二度と見せない側。詳細の応答は作成時のシークレットを運ばない。
	if strings.Contains(detail.Body.String(), created.ClientSecret) {
		t.Fatalf("detail response repeats the client secret: %s", detail.Body.String())
	}

	assignments := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID+"/assignments", csrf, cookie, nil)
	if assignments.Code != http.StatusOK || !strings.Contains(assignments.Body.String(), "alice") {
		t.Fatalf("assignments status=%d body=%s, want the saved assignment", assignments.Code, assignments.Body.String())
	}

	listed := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications", csrf, cookie, nil)
	var list struct {
		Applications []struct {
			ID string `json:"id"`
		} `json:"applications"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v; body=%s", err, listed.Body.String())
	}
	if len(list.Applications) != 1 || list.Applications[0].ID != applicationID {
		t.Fatalf("list=%+v, want only this tenant's application", list.Applications)
	}

	deleted := adminJSON(t, e, http.MethodDelete, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	if deleted.Code != http.StatusNoContent && deleted.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}

	for _, want := range []string{"ApplicationCreated", "ApplicationAssigned", "ApplicationDeleted"} {
		if !events.contains(want) {
			t.Fatalf("events=%v, want %s", events.types(), want)
		}
	}
}

//spec:covers REQ-APPLICATION-008, EX-APPLICATION-008-01: 上限内の画像のアップロードが `icon_object_key` と IdP の配信 URL を指す `icon_url` を持たせ、その URL が画像そのものを返し、削除で双方が空に戻ること。
func TestApplicationIconUploadExposesADeliveryURLAndDeleteClearsIt(t *testing.T) {
	e, _, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})

	uploaded := adminMultipart(t, e, "/api/admin/v1/applications/"+applicationID+"/icon", csrf, cookie, "icon.gif", gifBytes)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploaded.Code, uploaded.Body.String())
	}
	icon := readIconFields(t, uploaded.Body.Bytes()).Application
	if icon.IconObjectKey == "" {
		t.Fatalf("icon_object_key is empty: %s", uploaded.Body.String())
	}
	// 「IdP の配信 URL を指す」は、その URL が実際に画像を返すことでしか観測できない。
	if !strings.HasPrefix(icon.IconURL, "/realms/default/application-icons/"+applicationID+"/") {
		t.Fatalf("icon_url=%q, want an IdP delivery URL under this realm", icon.IconURL)
	}
	served := httptest.NewRequest(http.MethodGet, icon.IconURL, http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, served)
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), gifBytes) {
		t.Fatalf("icon fetch status=%d body=%v", response.Code, response.Body.Bytes())
	}

	// 管理一覧と利用者ポータルが同じ URL を読めること。
	listed := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications", csrf, cookie, nil)
	if !strings.Contains(listed.Body.String(), icon.IconURL) {
		t.Fatalf("admin list carries no icon_url: %s", listed.Body.String())
	}

	deleted := adminJSON(t, e, http.MethodDelete, "/api/admin/v1/applications/"+applicationID+"/icon", csrf, cookie, nil)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete icon status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	cleared := readIconFields(t, deleted.Body.Bytes()).Application
	if cleared.IconURL != "" || cleared.IconObjectKey != "" {
		t.Fatalf("icon fields not cleared: %+v", cleared)
	}
}

//spec:covers REQ-APPLICATION-008, EX-APPLICATION-008-02: 画像でないファイルのアップロードが 400 で拒否され、先に登録済みのアイコンの `icon_object_key` も配信されるバイト列も置き換わらないこと。
func TestApplicationIconRejectsANonImageAndKeepsTheExistingOne(t *testing.T) {
	e, _, _ := newApplicationHandlerRecordingEvents(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})
	uploaded := adminMultipart(t, e, "/api/admin/v1/applications/"+applicationID+"/icon", csrf, cookie, "icon.png", pngBytes)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploaded.Code, uploaded.Body.String())
	}
	original := readIconFields(t, uploaded.Body.Bytes()).Application

	refused := adminMultipart(t, e, "/api/admin/v1/applications/"+applicationID+"/icon", csrf, cookie, "payload.txt", []byte("not an image at all"))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
	}

	// 拒否が防いだ効果。保存済みのアイコンは鍵も内容も変わらない。
	detail := adminJSON(t, e, http.MethodGet, "/api/admin/v1/applications/"+applicationID, csrf, cookie, nil)
	kept := readIconFields(t, detail.Body.Bytes()).Application
	if kept.IconObjectKey != original.IconObjectKey || kept.IconURL != original.IconURL {
		t.Fatalf("icon replaced after the refusal: %+v, want %+v", kept, original)
	}
	served := httptest.NewRequest(http.MethodGet, original.IconURL, http.NoBody)
	response := httptest.NewRecorder()
	e.ServeHTTP(response, served)
	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), pngBytes) {
		t.Fatalf("stored icon changed: status=%d body=%v", response.Code, response.Body.Bytes())
	}
}

//spec:covers REQ-APPLICATION-013, EX-APPLICATION-013-01: admin ロールを持たない認証済み利用者の ListAdminApplications が 403 access_denied で拒否され、応答がテナントの Application を 1 つも運ばないこと。
func TestListAdminApplicationsRefusesAUserWithoutTheAdminRole(t *testing.T) {
	e := newApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)

	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})

	request := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/applications", http.NoBody)
	request.Header.Set("X-Demo-Sub", "regular")
	request.AddCookie(cookie)
	refused := httptest.NewRecorder()
	e.ServeHTTP(refused, request)

	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(refused.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, refused.Body.String())
	}
	if problem.Type != "urn:idmagic:error:access_denied" {
		t.Fatalf("problem type=%q, want the access_denied refusal", problem.Type)
	}
	// 拒否が防いだ効果。カタログの中身は応答に現れない。
	if body := refused.Body.String(); strings.Contains(body, applicationID) || strings.Contains(body, "Payroll") {
		t.Fatalf("refusal leaked the catalog: %s", body)
	}
}
