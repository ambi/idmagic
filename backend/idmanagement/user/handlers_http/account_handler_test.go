package handlers_http_test

// 主要ユースケース追跡: REQ-IDMANAGEMENT-016。

// SCL scenario "認証済みユーザーは自身のプロフィールを読み・編集できる" を
// /api/account/v1/profile 経由で検証する (wi-19)。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userhttp "github.com/ambi/idmagic/backend/idmanagement/user/handlers_http"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func newAccountServer(t *testing.T, user *userdomain.User) *echo.Echo {
	t.Helper()
	e, _ := newAccountServerWithRepo(t, user)
	return e
}

// newAccountServerWithRepo は保存層を呼び出し側へ返す。応答は use case の戻り値から
// 組み立てられるので、応答だけを読むテストは保存しない実装を通してしまう。自己サービスの
// 具体例はここから User を読み直して確かめる。`others` は認証されない同居利用者で、
// 「自分のデータだけを含む」を名指すために要る。
func newAccountServerWithRepo(
	t *testing.T, user *userdomain.User, others ...*userdomain.User,
) (*echo.Echo, *usermemory.UserRepository) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if user != nil {
		userRepo.Seed(user)
	}
	for _, other := range others {
		userRepo.Seed(other)
	}
	tenantRepo := tenancymemory.NewTenantRepository()
	if err := tenantRepo.Save(context.Background(), activeTenant(tenancydomain.DefaultTenantID, "Default")); err != nil {
		t.Fatal(err)
	}
	resolver := &fakeAuthnResolver{}
	if user != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			UserID: user.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenantRepo,
		Emit:       func(spec.DomainEvent) {}, UserRepo: userRepo,
		AttrSchemaRepo: usermemory.NewTenantUserAttributeSchemaRepository(),
		AuthnResolver:  resolver,
	})
	return e, userRepo
}

func accountUser() *userdomain.User {
	now := time.Now().UTC()
	name := "Dave Q"
	return &userdomain.User{
		ID: "user-1", PreferredUsername: "dave", TenantID: tenancydomain.DefaultTenantID, Name: &name,
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$aGFzaGhhc2g",
		Lifecycle:    userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		Attributes: map[string]userdomain.AttributeValue{
			"nickname":   {Type: idmdomain.AttributeTypeString, String: new("davey")},    // claim_exposed
			"department": {Type: idmdomain.AttributeTypeString, String: new("Platform")}, // self_readable
		},
		CreatedAt: now, UpdatedAt: now,
	}
}

func TestAccountProfileGetRequiresAuth(t *testing.T) {
	e := newAccountServer(t, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/profile", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAccountProfileGetReturnsSelfView(t *testing.T) {
	e := newAccountServer(t, accountUser())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/profile", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body userhttp.AccountProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body.Attributes["nickname"]; !ok {
		t.Fatalf("claim_exposed nickname missing from self view: %+v", body.Attributes)
	}
	if v, ok := body.Attributes["department"]; !ok || v.String == nil || *v.String != "Platform" {
		t.Fatalf("self_readable department missing from self view: %+v", body.Attributes)
	}
	if len(body.ReadableAttributes) == 0 {
		t.Fatalf("readable_attributes should be populated")
	}
	if len(body.EditableAttributes) == 0 {
		t.Fatalf("editable_attributes should be populated")
	}
}

func TestAccountSummaryRequiresAuth(t *testing.T) {
	e := newAccountServer(t, nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/summary", http.NoBody))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// **「他人のものが混じらない」を、同居利用者を置いて名指す。** 利用者が 1 人しか
// 居ないテナントでは、対象を取り違える実装でも同じ応答になってしまう。
//
//spec:covers EX-IDMANAGEMENT-019-01: アカウント概要が返すのは呼び出し元自身のデータだけで、ロールを含まないこと。
func TestAccountSummaryReturnsLifecycleAndOmitsRoles(t *testing.T) {
	user := accountUser()
	last := time.Date(2026, 6, 20, 9, 0, 0, 0, time.UTC)
	user.Lifecycle.LastLoginAt = &last
	user.Lifecycle.RequiredActions = []idmdomain.RequiredAction{idmdomain.RequiredActionUpdatePassword}
	user.Roles = []string{"admin"}
	e, _ := newAccountServerWithRepo(t, user, accountOtherUser())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/summary", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["roles"]; ok {
		t.Fatalf("summary must not expose roles: %+v", body)
	}
	// 同居利用者ではなく呼び出し元のものが返っていること。
	if body["id"] != user.ID || body["preferred_username"] != user.PreferredUsername {
		t.Fatalf("summary is not the caller's: %+v", body)
	}
	if body["last_login_at"] == nil {
		t.Fatalf("last_login_at missing: %+v", body)
	}
	actions, ok := body["required_actions"].([]any)
	if !ok || len(actions) != 1 || actions[0] != string(idmdomain.RequiredActionUpdatePassword) {
		t.Fatalf("required_actions not projected: %+v", body["required_actions"])
	}
}

// accountOtherUser は認証されない同居利用者。自己サービスの応答が呼び出し元の
// ものであることを、対象の取り違えが観測できる形にするために要る。
func accountOtherUser() *userdomain.User {
	now := time.Now().UTC()
	name := "Erin R"
	return &userdomain.User{
		ID: "user-2", PreferredUsername: "erin", TenantID: tenancydomain.DefaultTenantID, Name: &name,
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$aGFzaGhhc2g",
		Lifecycle:    userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt:    now, UpdatedAt: now,
	}
}

func TestAccountProfilePatchUpdatesEditableAttribute(t *testing.T) {
	e := newAccountServer(t, accountUser())
	rec := patchSettings(t, e, map[string]any{
		"given_name": "Dave",
		"attributes": map[string]any{
			"nickname": map[string]any{"type": "string", "string": "newnick"},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body userhttp.AccountProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.GivenName == nil || *body.GivenName != "Dave" {
		t.Fatalf("given_name not updated: %+v", body.GivenName)
	}
	if v := body.Attributes["nickname"]; v.String == nil || *v.String != "newnick" {
		t.Fatalf("nickname not updated: %+v", body.Attributes)
	}
}

// 具体例が言う 2 つの `Then` に 2 つの観測を置く。どちらも保存層から読み直す。
// 応答は use case の戻り値から組み立てられるので、表示名だけを応答で見ると保存しない
// 実装が通り、拒否だけを status で見ると拒否のついでに書いてしまう実装が通る。
//
//spec:covers EX-IDMANAGEMENT-016-01: 表示名の更新が保存されること、editable_by_user=false の属性が同じ入口では更新できないこと。
func TestAccountProfileUpdatesDisplayNameButNotAdminManagedAttributes(t *testing.T) {
	user := accountUser()
	e, users := newAccountServerWithRepo(t, user)

	renamed := patchSettings(t, e, map[string]any{
		"name": "Dave Renamed",
	})
	if renamed.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", renamed.Code, renamed.Body.String())
	}
	stored, err := users.FindBySub(context.Background(), user.ID)
	if err != nil || stored == nil {
		t.Fatalf("FindByID=(%v,%v)", stored, err)
	}
	if stored.Name == nil || *stored.Name != "Dave Renamed" {
		t.Fatalf("表示名が保存されていない: %v", stored.Name)
	}

	// `department` は org 属性で `editable_by_user=false`。自己サービスからは動かない。
	refused := patchSettings(t, e, map[string]any{
		"attributes": map[string]any{
			"department": map[string]any{"type": "string", "string": "Sales"},
		},
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	stored, err = users.FindBySub(context.Background(), user.ID)
	if err != nil || stored == nil {
		t.Fatalf("FindByID=(%v,%v)", stored, err)
	}
	if got := stored.Attributes["department"]; got.String == nil || *got.String != "Platform" {
		t.Fatalf("拒否されたのに department が変わった: %+v", got)
	}
}

// docs/design/application/api-rules.md は 400 を「リクエストを解析できない」、422 を「解析できた内容が
// 業務規則に違反する」と定める。テナントの属性スキーマへの適合は後者なので、
// 属性スキーマ違反は 422 で返る。同じ違反を UpdateAdminUser は既に 422 で返しており、
// 契約 (UpdateUserProfileError422 / InvalidUserAttributeError) もそちらを書いている。
func TestAccountProfilePatchRejectsSchemaViolationAsUnprocessable(t *testing.T) {
	e := newAccountServer(t, accountUser())
	rec := patchSettings(t, e, map[string]any{
		"attributes": map[string]any{
			// 実効スキーマ (組み込み ∪ tenant) に無い key。解析はできるが業務規則に反する。
			"zone": map[string]any{"type": "string", "string": "z"},
		},
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s, want 422", rec.Code, rec.Body.String())
	}
	var problem struct {
		Type   string `json:"type"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v (body=%s)", err, rec.Body.String())
	}
	if problem.Type != "urn:idmagic:error:invalid_attribute" || problem.Status != http.StatusUnprocessableEntity {
		t.Fatalf("problem=%+v, want invalid_attribute with status 422", problem)
	}
	// 拒否が何も通していないこと。読み戻して属性が増えていないことを確かめる。
	after := httptest.NewRecorder()
	e.ServeHTTP(after, httptest.NewRequest(http.MethodGet, "/realms/default/api/account/v1/profile", http.NoBody))
	var profile userhttp.AccountProfileResponse
	if err := json.Unmarshal(after.Body.Bytes(), &profile); err != nil {
		t.Fatalf("decode profile: %v (body=%s)", err, after.Body.String())
	}
	if _, stored := profile.Attributes["zone"]; stored {
		t.Fatalf("refused update stored the attribute: %+v", profile.Attributes)
	}
}

type fakeAuthnResolver struct {
	ctx *authdomain.AuthenticationContext
}

func (f *fakeAuthnResolver) Resolve(_ context.Context, _ authdomain.Headers) (*authdomain.AuthenticationContext, error) {
	return f.ctx, nil
}

func activeTenant(id, displayName string) *tenancydomain.Tenant {
	realm := id
	if id == tenancydomain.DefaultTenantID {
		realm = tenancydomain.DefaultRealm
	}
	return &tenancydomain.Tenant{
		ID: id, Realm: realm, DisplayName: displayName, Status: tenancydomain.TenantStatusActive,
		CreatedAt: time.Now().UTC(),
	}
}

// accountProfilePath は自己サービスのプロフィール入口。テナントは default 固定で、
// パスワード再設定の文脈から取った CSRF の組をそのまま使う。
const accountProfilePath = "/realms/default/api/account/v1/profile"

func patchSettings(t *testing.T, e *echo.Echo, body any) *httptest.ResponseRecorder {
	t.Helper()
	path := accountProfilePath
	csrf, cookie := passwordResetContextCSRF(
		t, e, tenantPrefix(path)+"/api/auth/password_reset_context",
	)
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func passwordResetContextCSRF(t *testing.T, e *echo.Echo, path string) (string, *http.Cookie) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("csrf context status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("csrf cookie missing")
	}
	return body.CSRFToken, cookies[0]
}

func tenantPrefix(path string) string {
	const prefix = "/realms/"
	if len(path) < len(prefix) || path[:len(prefix)] != prefix {
		return ""
	}
	rest := path[len(prefix):]
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			return prefix + rest[:i]
		}
	}
	return prefix + rest
}
