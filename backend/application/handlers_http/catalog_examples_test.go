package handlers_http_test

// Application カタログ (REQ-APPLICATION-007)、アイコン (REQ-APPLICATION-008)、
// 管理 API のロール境界 (REQ-APPLICATION-013) が宣言する具体例を、管理 API の入口で観測する。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/saml"
	samlmemory "github.com/ambi/idmagic/backend/saml/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedmemory "github.com/ambi/idmagic/backend/wsfederation/db_memory"

	"github.com/labstack/echo/v5"
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

//spec:covers EX-APPLICATION-007-03: 別テナントの管理者が同じ id で GetAdminApplication を呼ぶと、存在しない id と同じ 404 application_not_found になり、名称と OIDC 設定を返さないこと。
func TestGetAdminApplicationFromAnotherTenantIsNotFound(t *testing.T) {
	e := newTwoTenantApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "portal", "type": "oidc", "client_type": "confidential",
		"token_endpoint_auth_method": "client_secret_post",
		"redirect_uris":              []string{"https://portal.example/callback"},
	})

	// 同じテナントでは取得できることを先に確かめ、下の拒否が URL の誤りではないことを示す。
	if own := getAs(e, "admin", "/realms/default/api/admin/v1/applications/"+applicationID); own.Code != http.StatusOK {
		t.Fatalf("own tenant status=%d body=%s", own.Code, own.Body.String())
	}

	foreign := getAs(e, "globex-admin", "/realms/globex/api/admin/v1/applications/"+applicationID)
	missing := getAs(e, "globex-admin", "/realms/globex/api/admin/v1/applications/ffffffff-ffff-4fff-bfff-ffffffffffff")
	if foreign.Code != http.StatusNotFound || readProblemType(t, foreign) != "urn:idmagic:error:application_not_found" {
		t.Fatalf("foreign tenant status=%d body=%s, want 404 application_not_found", foreign.Code, foreign.Body.String())
	}
	assertIndistinguishableFromMissing(t, foreign, missing)
	if body := foreign.Body.String(); strings.Contains(body, "portal") || strings.Contains(body, applicationID) {
		t.Fatalf("foreign tenant response carries the application: %s", body)
	}
}

//spec:covers EX-APPLICATION-008-03: 別テナントの realm で同じ `application_id` と id のアイコンを取得すると、存在しない id と同じ 404 not_found になり、画像の内容を返さないこと。
func TestGetApplicationIconFromAnotherTenantIsNotFound(t *testing.T) {
	e := newTwoTenantApplicationHandler(t)
	csrf, cookie := appCSRF(t, e)
	applicationID := createApplication(t, e, csrf, cookie, map[string]any{
		"name": "Payroll", "type": "weblink", "launch_url": "https://payroll.example",
	})
	uploaded := adminMultipart(t, e, "/api/admin/v1/applications/"+applicationID+"/icon", csrf, cookie, "icon.gif", gifBytes)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploaded.Code, uploaded.Body.String())
	}
	objectKey := readIconFields(t, uploaded.Body.Bytes()).Application.IconObjectKey

	// アイコンの配信は認証を要求しないので、要求者ではなく realm だけがテナントを決める。
	iconPath := "/application-icons/" + applicationID + "/"
	if own := getAs(e, "", "/realms/default"+iconPath+objectKey); own.Code != http.StatusOK || !bytes.Equal(own.Body.Bytes(), gifBytes) {
		t.Fatalf("own tenant status=%d body=%v", own.Code, own.Body.Bytes())
	}

	foreign := getAs(e, "", "/realms/globex"+iconPath+objectKey)
	missing := getAs(e, "", "/realms/globex"+iconPath+"ffffffff-ffff-4fff-bfff-ffffffffffff")
	if foreign.Code != http.StatusNotFound || readProblemType(t, foreign) != "urn:idmagic:error:not_found" {
		t.Fatalf("foreign tenant status=%d body=%s, want 404 not_found", foreign.Code, foreign.Body.String())
	}
	assertIndistinguishableFromMissing(t, foreign, missing)
	if bytes.Contains(foreign.Body.Bytes(), gifBytes) {
		t.Fatalf("foreign tenant response carries the icon: %v", foreign.Body.Bytes())
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

// newTwoTenantApplicationHandler は、既定の fixture に globex テナントとその管理者を足す。
// テナントの登録先が無いと default 以外の realm は解決されず、越境の拒否を
// 存在しないテナントの拒否と区別して観測できない。
func newTwoTenantApplicationHandler(t *testing.T) *echo.Echo {
	t.Helper()
	now := time.Now().UTC()
	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: "globex", Realm: "globex", DisplayName: "Globex", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	users := usermemory.NewUserRepository()
	users.Seed(&userdomain.User{
		ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	users.Seed(&userdomain.User{
		ID: "globex-admin", TenantID: "globex", PreferredUsername: "admin",
		PasswordHash: "unused", Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:     "http://idp.test",
		Emit:       func(spec.DomainEvent) {},
		TenantRepo: tenants, UserRepo: users, GroupRepo: groupmemory.NewGroupRepository(),
		Application: application.Module{
			Repo:                    appmemory.NewApplicationRepository(),
			IconStore:               appmemory.NewApplicationIconStore(),
			AssignmentRepo:          appmemory.NewApplicationAssignmentRepository(),
			OrderingRepo:            appmemory.NewApplicationOrderingRepository(),
			CategoryRepo:            appmemory.NewApplicationCategoryRepository(),
			SignInPolicyRepo:        appmemory.NewSignInPolicyRepository(),
			DefaultSignInPolicyRepo: appmemory.NewDefaultSignInPolicyRepository(),
		},
		Saml:          saml.Module{SPRepo: samlmemory.NewSamlServiceProviderRepository()},
		OAuth2:        oauth2.Module{ClientRepo: oauth2memory.NewClientRepository()},
		WsFederation:  wsfederation.Module{RPRepo: wsfedmemory.NewWsFedRelyingPartyRepository()},
		AuthnResolver: authusecases.DemoHeaderResolver{},
	})
	return e
}

// getAs は sub として認証した GET を送る。sub が空なら認証しない。
func getAs(e *echo.Echo, sub, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	if sub != "" {
		request.Header.Set("X-Demo-Sub", sub)
	}
	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)
	return response
}

func readProblemType(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var problem struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, response.Body.String())
	}
	return problem.Type
}

// assertIndistinguishableFromMissing は、越境の応答が存在しない id の応答と区別できないことを表明する。
// 区別できれば、別テナントに同じ id があることを推測させる。
func assertIndistinguishableFromMissing(t *testing.T, foreign, missing *httptest.ResponseRecorder) {
	t.Helper()
	if foreign.Code != missing.Code || foreign.Body.String() != missing.Body.String() ||
		foreign.Header().Get("Content-Type") != missing.Header().Get("Content-Type") {
		t.Fatalf("foreign tenant response differs from a missing id:\nforeign=%d %s\nmissing=%d %s",
			foreign.Code, foreign.Body.String(), missing.Code, missing.Body.String())
	}
}
