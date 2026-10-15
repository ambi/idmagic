package handlers_http_test

// 主要ユースケース追跡: REQ-TENANCY-004。

// SCL scenario "管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に
// 反映される" / "不正な branding 入力は拒否されシステム既定にフォールバックする" を
// /api/branding, /api/admin/v1/tenant/branding, /api/admin/v1/tenant/branding/assets/{kind}
// 経由で検証する (wi-89)。GetTenantBranding / GetTenantBrandingAsset は
// public、更新系のみ tenant admin に制限される。

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	memory "github.com/ambi/idmagic/backend/tenancy/internal/db_memory"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/internal/handlers_http"

	"github.com/labstack/echo/v5"
)

func newBrandingServer(t *testing.T, actor *userdomain.User, tenants ...*domain.Tenant) (*echo.Echo, *memory.TenantBrandingRepository, *memory.TenantBrandingAssetStore, *[]spec.DomainEvent) {
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
	brandingRepo := memory.NewTenantBrandingRepository()
	assetStore := memory.NewTenantBrandingAssetStore()
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
		Issuer: "http://idp.test", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenantRepo,
		Emit:       emit,
		Tenancy: tenancy.Module{
			TenantRepo:         tenantRepo,
			BrandingRepo:       brandingRepo,
			BrandingAssetStore: assetStore,
		},
		UserRepo:      userRepo,
		AuthnResolver: resolver,
	})
	return e, brandingRepo, assetStore, &events
}

func TestGetBrandingReturnsEmptyForUnconfiguredTenant(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, nil, activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/branding", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body tenancyhttp.BrandingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ProductName != "" || body.PrimaryColor != "" || body.UpdatedAt != nil {
		t.Fatalf("expected empty branding, got %+v", body)
	}
}

func TestGetBrandingSupportsIfNoneMatch(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, nil, activeTenant("acme", "Acme"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/branding", http.NoBody))
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatal("expected ETag header")
	}
	req := httptest.NewRequest(http.MethodGet, "/realms/acme/api/branding", http.NoBody)
	req.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("status=%d want 304", rec2.Code)
	}
}

//verify:operation UpdateTenantBranding: 非管理者の設定変更は 403 access_denied の Problem Details で拒否する。
//verify:operation UploadTenantBrandingAsset: 非管理者の画像アップロードは 403 access_denied の Problem Details で拒否する。
//verify:operation DeleteTenantBrandingAsset: 非管理者の画像削除は 403 access_denied の Problem Details で拒否する。
func TestUpdateBrandingRejectsNonAdmin(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, settingsActor("alice", "acme", nil), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{"product_name": "Acme"})
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	assertTenancyProblem(t, resp, http.StatusForbidden, "access_denied")
	assertTenancyProblem(t, uploadBrandingAsset(t, e, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}), http.StatusForbidden, "access_denied")
	assertTenancyProblem(t, deleteBrandingAsset(t, e, "/realms/acme/api/admin/v1/tenant/branding/assets/logo"), http.StatusForbidden, "access_denied")
}

//spec:covers EX-TENANCY-004-01: 管理者が設定した配色、footer link、footer text は保存され、公開 branding から読める。
//verify:operation UpdateTenantBranding: 200 JSON は保存した製品名、色、フッターを返し、公開 GET に反映される。
//verify:operation GetTenantBranding: 公開 GET は 200 JSON に保存した製品名、色、フッターを返す。
func TestUpdateBrandingPersistsAndIsPubliclyVisible(t *testing.T) {
	e, repo, _, events := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{
		"product_name":  "Acme",
		"primary_color": "#0f172a",
		"accent_color":  "#38bdf8",
		"footer_link_1": map[string]any{"label": "ヘルプ", "url": "https://support.example.com"},
		"footer_text":   "(c) Acme",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	saved, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || saved == nil || saved.ProductName != "Acme" {
		t.Fatalf("branding not persisted: %+v err=%v", saved, err)
	}
	if len(*events) != 1 {
		t.Fatalf("events=%d want 1", len(*events))
	}
	if _, ok := (*events)[0].(*domain.TenantBrandingUpdated); !ok {
		t.Fatalf("event type=%T", (*events)[0])
	}

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/realms/acme/api/branding", http.NoBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body tenancyhttp.BrandingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.ProductName != "Acme" || body.PrimaryColor != "#0f172a" || body.AccentColor != "#38bdf8" ||
		body.FooterLink1 == nil || body.FooterLink1.Label != "ヘルプ" || body.FooterText != "(c) Acme" {
		t.Fatalf("public branding not reflected: %+v", body)
	}
	var updated tenancyhttp.BrandingResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.ProductName != body.ProductName || updated.PrimaryColor != body.PrimaryColor || updated.AccentColor != body.AccentColor || updated.FooterText != body.FooterText || updated.FooterLink1 == nil || updated.FooterLink1.Label != body.FooterLink1.Label || updated.FooterLink1.URL != "https://support.example.com" {
		t.Fatalf("updated branding=%+v, public=%+v", updated, body)
	}
}

func TestUpdateBrandingRejectsNonHTTPSSupportURL(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{
		"footer_link_1": map[string]any{"label": "ヘルプ", "url": "javascript:alert(1)"},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("invalid_branding")) {
		t.Fatalf("unexpected body=%s", resp.Body.String())
	}
}

func TestUpdateBrandingRejectsIncompleteFooterLink(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{
		"footer_link_1": map[string]any{"label": "ヘルプ"},
	})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestUpdateBrandingAcceptsLowContrastColor(t *testing.T) {
	e, repo, _, _ := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{
		"primary_color": "#eeeeee",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	saved, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || saved == nil || saved.PrimaryColor != "#eeeeee" {
		t.Fatalf("low-contrast color was not persisted: %+v err=%v", saved, err)
	}
}

//spec:covers EX-TENANCY-004-01: PNG ロゴの upload response が示す同一 realm の URL から検証済み画像を取得できる。
//verify:operation UploadTenantBrandingAsset: 200 JSON の branding.logo_url は保存した画像を自身のテナントから配信する URL を返す。
//verify:operation DeleteTenantBrandingAsset: 200 JSON の branding は logo_url を省き、保存した画像参照も消す。
func TestUploadAndDeleteBrandingLogoAsset(t *testing.T) {
	e, repo, assetStore, events := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}

	uploadResp := uploadBrandingAsset(t, e, png)
	if uploadResp.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploadResp.Code, uploadResp.Body.String())
	}
	saved, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || saved == nil || saved.LogoObjectKey == "" {
		t.Fatalf("logo reference not persisted: %+v err=%v", saved, err)
	}
	uploadBody := decodeJSON(t, uploadResp)
	uploadBranding, ok := uploadBody["branding"].(map[string]any)
	if !ok || uploadBranding["logo_url"] != saved.LogoURL {
		t.Fatalf("upload response=%v, want logo_url=%s", uploadBody, saved.LogoURL)
	}
	if _, err := assetStore.Find(context.Background(), "acme", domain.TenantBrandingAssetKindLogo, saved.LogoObjectKey); err != nil {
		t.Fatalf("find asset: %v", err)
	}

	getRec := httptest.NewRecorder()
	e.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, saved.LogoURL, http.NoBody))
	if getRec.Code != http.StatusOK {
		t.Fatalf("asset get status=%d body=%s", getRec.Code, getRec.Body.String())
	}
	if getRec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("expected nosniff header on asset serving")
	}

	deleteResp := deleteBrandingAsset(t, e, "/realms/acme/api/admin/v1/tenant/branding/assets/logo")
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	deleteBody := decodeJSON(t, deleteResp)
	deleteBranding, ok := deleteBody["branding"].(map[string]any)
	if !ok {
		t.Fatalf("delete response=%v, want branding object", deleteBody)
	}
	if _, exists := deleteBranding["logo_url"]; exists {
		t.Fatalf("delete branding retains logo_url: %v", deleteBranding)
	}
	cleared, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || cleared == nil || cleared.LogoObjectKey != "" {
		t.Fatalf("expected logo cleared: %+v err=%v", cleared, err)
	}
	if len(*events) != 2 {
		t.Fatalf("events=%d want 2 (upload + delete)", len(*events))
	}
}

//spec:covers EX-TENANCY-004-02: 別 realm で同じ kind と id を取得すると、存在しない id と同じ 404 not_found になり PNG の内容を返さない。
func TestGetBrandingAssetFromAnotherRealmIsNotFound(t *testing.T) {
	e, repo, _, _ := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}),
		activeTenant("acme", "Acme"), activeTenant("globex", "Globex"))
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 'a', 'c', 'm', 'e'}

	uploadResp := uploadBrandingAsset(t, e, png)
	if uploadResp.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploadResp.Code, uploadResp.Body.String())
	}
	saved, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || saved == nil || saved.LogoObjectKey == "" {
		t.Fatalf("logo reference not persisted: %+v err=%v", saved, err)
	}

	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))
		return rec
	}
	// 同じ realm では取得できることを先に確かめ、下の拒否が URL の誤りではないことを示す。
	if own := get("/realms/acme/tenant-branding-assets/logo/" + saved.LogoObjectKey); own.Code != http.StatusOK || !bytes.Equal(own.Body.Bytes(), png) {
		t.Fatalf("own realm status=%d body=%q", own.Code, own.Body.Bytes())
	}

	foreign := get("/realms/globex/tenant-branding-assets/logo/" + saved.LogoObjectKey)
	missing := get("/realms/globex/tenant-branding-assets/logo/00000000-0000-4000-8000-000000000000")
	if foreign.Code != http.StatusNotFound {
		t.Fatalf("foreign realm status=%d body=%s", foreign.Code, foreign.Body.String())
	}
	// 別テナントに同じ id があるかを推測させないため、未存在の id と区別できない応答にする。
	if foreign.Code != missing.Code || foreign.Body.String() != missing.Body.String() ||
		foreign.Header().Get("Content-Type") != missing.Header().Get("Content-Type") {
		t.Fatalf("foreign realm response differs from a missing id:\nforeign=%d %s\nmissing=%d %s",
			foreign.Code, foreign.Body.String(), missing.Code, missing.Body.String())
	}
	if bytes.Contains(foreign.Body.Bytes(), png) {
		t.Fatalf("foreign realm response carries the asset: %q", foreign.Body.Bytes())
	}
}

func TestUploadBrandingAssetRejectsSVG(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := uploadBrandingAsset(t, e, []byte("<svg onload=alert(1)></svg>"))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func patchBranding(t *testing.T, e *echo.Echo, body any) *httptest.ResponseRecorder {
	t.Helper()
	const path = "/realms/acme/api/admin/v1/tenant/branding"
	csrf, cookie := passwordResetContextCSRF(t, e, "/realms/acme/api/auth/password-reset-context")
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func uploadBrandingAsset(t *testing.T, e *echo.Echo, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	const path = "/realms/acme/api/admin/v1/tenant/branding/assets/logo"
	csrf, cookie := passwordResetContextCSRF(t, e, tenantPrefix(path)+"/api/auth/password-reset-context")
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func deleteBrandingAsset(t *testing.T, e *echo.Echo, path string) *httptest.ResponseRecorder {
	t.Helper()
	csrf, cookie := passwordResetContextCSRF(t, e, tenantPrefix(path)+"/api/auth/password-reset-context")
	req := httptest.NewRequest(http.MethodDelete, path, http.NoBody)
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
