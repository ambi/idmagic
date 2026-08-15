package handlers_http_test

// SCL scenario "テナント内 admin は所属テナントの設定を読み・更新できる"
// を /api/admin/v1/settings 経由で検証する。AdminSettingsRead は admin /
// system_admin の両方で許可、AdminSettingsUpdate は actor.tenant_id に
// 固定する。password_policy_override の弱化は use case 側で reject される。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/handlers_http"

	"github.com/labstack/echo/v5"
)

func newSettingsServer(t *testing.T, actor *userdomain.User, tenants ...*domain.Tenant) (*echo.Echo, *memory.TenantRepository, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	tenantRepo := memory.NewTenantRepository()
	for _, tenant := range tenants {
		if err := tenantRepo.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	events := make([]spec.DomainEvent, 0)
	emit := func(e spec.DomainEvent) { events = append(events, e) }
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Deps: support.Deps{
			Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
			TenantRepo: tenantRepo,
			Emit:       emit,
		}, UserRepo: userRepo,

		AuthnResolver: resolver,
	})
	return e, tenantRepo, &events
}

func settingsActor(sub, tenantID string, roles []string) *userdomain.User {
	now := time.Now().UTC()
	return &userdomain.User{
		ID: sub, PreferredUsername: sub, TenantID: tenantID, Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

func activeTenant(id, displayName string) *domain.Tenant {
	realm := id
	if id == domain.DefaultTenantID {
		realm = domain.DefaultRealm
	}
	return &domain.Tenant{
		ID: id, Realm: realm, DisplayName: displayName, Status: domain.TenantStatusActive,
		CreatedAt: time.Now().UTC(),
	}
}

func TestAdminSettingsGetRejectsNonAdmin(t *testing.T) {
	e, _, _ := newSettingsServer(t, settingsActor("alice", "acme", nil), activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/settings", http.NoBody))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminSettingsGetReturnsCurrentTenant(t *testing.T) {
	minLength := 16
	tenant := activeTenant("acme", "Acme")
	tenant.PasswordPolicyOverride = &domain.PasswordPolicyOverride{MinLength: &minLength}
	e, _, _ := newSettingsServer(t, settingsActor("admin", "acme", []string{"admin"}), tenant)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/settings", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body tenancyhttp.AdminSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.TenantID != "acme" || body.DisplayName != "Acme" {
		t.Fatalf("body=%+v", body)
	}
	if body.PasswordPolicyOverride == nil || body.PasswordPolicyOverride.MinLength == nil ||
		*body.PasswordPolicyOverride.MinLength != minLength {
		t.Fatalf("override=%+v", body.PasswordPolicyOverride)
	}
	if body.PasswordPolicyDefaults.MinLength <= 0 ||
		body.PasswordPolicyDefaults.MaxLength <= 0 ||
		body.PasswordPolicyDefaults.HistoryDepth <= 0 {
		t.Fatalf("defaults must be populated: %+v", body.PasswordPolicyDefaults)
	}
}

func TestAdminSettingsGetAllowsSystemAdmin(t *testing.T) {
	e, _, _ := newSettingsServer(
		t,
		settingsActor("ops", domain.DefaultTenantID, []string{"system_admin"}),
		activeTenant(domain.DefaultTenantID, "Default"),
	)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/settings", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminSettingsPatchUpdatesAndEmitsEvent(t *testing.T) {
	e, repo, events := newSettingsServer(
		t,
		settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"),
	)
	resp := patchSettings(t, e, map[string]any{
		"display_name": "Acme Inc.",
		"password_policy_override": map[string]int{
			"min_length": 16, "history_depth": 10,
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	tenant, err := repo.FindByID(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if tenant.DisplayName != "Acme Inc." {
		t.Fatalf("display_name=%q", tenant.DisplayName)
	}
	if tenant.PasswordPolicyOverride == nil ||
		tenant.PasswordPolicyOverride.MinLength == nil ||
		*tenant.PasswordPolicyOverride.MinLength != 16 {
		t.Fatalf("override=%+v", tenant.PasswordPolicyOverride)
	}
	if len(*events) != 1 {
		t.Fatalf("events=%d want 1", len(*events))
	}
	updated, ok := (*events)[0].(*domain.TenantUpdated)
	if !ok {
		t.Fatalf("event type=%T", (*events)[0])
	}
	if updated.TenantID != "acme" {
		t.Fatalf("event tenant=%q", updated.TenantID)
	}
}

// REQ-TENANCY-021: 設定 API は現在の上書きとシステム上限を返し、厳しい上書きだけを保存する。
func TestAdminSettingsExposeAndUpdateMaxDelegationDepth(t *testing.T) {
	e, repo, _ := newSettingsServer(
		t,
		settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"),
	)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/settings", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var initial tenancyhttp.AdminSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.MaxDelegationDepth != nil || initial.MaxDelegationDepthDefault != domain.DefaultMaxDelegationDepth {
		t.Fatalf("delegation settings=%+v", initial)
	}

	resp := patchSettings(t, e, map[string]any{"max_delegation_depth": 1})
	if resp.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", resp.Code, resp.Body.String())
	}
	stored, err := repo.FindByID(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if stored.MaxDelegationDepth == nil || *stored.MaxDelegationDepth != 1 {
		t.Fatalf("stored max_delegation_depth=%v", stored.MaxDelegationDepth)
	}

	resp = patchSettings(t, e, map[string]any{
		"max_delegation_depth": domain.DefaultMaxDelegationDepth + 1,
	})
	if resp.Code != http.StatusBadRequest || !bytes.Contains(resp.Body.Bytes(), []byte("policy_override_weaker")) {
		t.Fatalf("weaker override status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAdminSettingsPatchRejectsWeakerPolicy(t *testing.T) {
	e, _, _ := newSettingsServer(
		t,
		settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"),
	)
	resp := patchSettings(t, e, map[string]any{
		"password_policy_override": map[string]int{"min_length": 4},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("policy_override_weaker")) {
		t.Fatalf("unexpected body=%s", resp.Body.String())
	}
}

// admin が /realms/{自テナント}/api/admin/v1/settings から触れるのは自テナントのみで、
// 別テナントを書き換える経路は存在しない。
func TestAdminSettingsPatchStaysWithinActorTenant(t *testing.T) {
	e, repo, _ := newSettingsServer(
		t,
		settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"),
		activeTenant("other", "Other"),
	)
	resp := patchSettings(t, e, map[string]any{
		"display_name": "Modified",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	other, err := repo.FindByID(context.Background(), "other")
	if err != nil {
		t.Fatal(err)
	}
	if other.DisplayName != "Other" {
		t.Fatalf("other tenant was modified: %q", other.DisplayName)
	}
}

func TestAdminSettingsPatchRequiresCSRF(t *testing.T) {
	e, _, _ := newSettingsServer(
		t,
		settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"),
	)
	body, err := json.Marshal(map[string]string{"display_name": "X"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPatch, "/realms/acme/api/admin/v1/settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Fatalf("status=%d (CSRF should reject)", rec.Code)
	}
}

// patchSettings は /api/admin/v1/settings への PATCH。この endpoint は解決済みテナントに
// 固定されるため、path は呼び出し側で選ぶ余地が無い。
func patchSettings(t *testing.T, e *echo.Echo, body any) *httptest.ResponseRecorder {
	t.Helper()
	const path = "/realms/acme/api/admin/v1/settings"
	// CSRF token / cookie を tenant local の password_reset_context 経由で発行する。
	tenant := tenantPrefix(path)
	csrf, cookie := passwordResetContextCSRF(t, e, tenant+"/api/auth/password_reset_context")
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

// テナント既定 locale は通知の locale 解決の第 2 段。UI が選択肢を
// 組み立てられるよう、同梱翻訳を持つ locale 一覧も同じレスポンスで返す。
func TestAdminSettingsExposeDefaultLocale(t *testing.T) {
	e, repo, _ := newSettingsServer(
		t,
		settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"),
	)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/admin/v1/settings", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var initial tenancyhttp.AdminSettingsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &initial); err != nil {
		t.Fatal(err)
	}
	if initial.DefaultLocale != "" {
		t.Errorf("default_locale=%q, want empty (system default)", initial.DefaultLocale)
	}
	if len(initial.SupportedLocales) < 2 {
		t.Errorf("supported_locales=%v, want at least ja and en", initial.SupportedLocales)
	}

	if resp := patchSettings(t, e, map[string]any{
		"default_locale": "ja",
	}); resp.Code != http.StatusOK {
		t.Fatalf("patch status=%d body=%s", resp.Code, resp.Body.String())
	}
	tenant, err := repo.FindByID(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if tenant.DefaultLocale == nil || *tenant.DefaultLocale != "ja" {
		t.Fatalf("stored default_locale=%v, want ja", tenant.DefaultLocale)
	}

	// 同梱翻訳を持たない locale は拒否する。通知が空の本文で届くより先に落とす。
	if resp := patchSettings(t, e, map[string]any{
		"default_locale": "fr",
	}); resp.Code != http.StatusBadRequest {
		t.Fatalf("unsupported locale status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func tenantPrefix(path string) string {
	// "/realms/acme/api/admin/v1/settings" -> "/realms/acme"
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
