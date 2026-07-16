package http_test

// SCL scenario "管理者はテナントのロゴと配色をカスタマイズでき利用者のログイン画面に
// 反映される" / "不正な branding 入力は拒否されシステム既定にフォールバックする" を
// /api/branding, /api/admin/tenant/branding, /api/admin/tenant/branding/assets/{kind}
// 経由で検証する (wi-89, ADR-096)。GetTenantBranding / GetTenantBrandingAsset は
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
	idmmemory "github.com/ambi/idmagic/backend/identitymanagement/adapters/persistence/memory"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/adapters/http/server"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancyhttp "github.com/ambi/idmagic/backend/tenancy/adapters/http"
	"github.com/ambi/idmagic/backend/tenancy/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func newBrandingServer(t *testing.T, actor *idmdomain.User, tenants ...*domain.Tenant) (*echo.Echo, *memory.TenantBrandingRepository, *memory.TenantBrandingAssetStore, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := idmmemory.NewUserRepository()
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
		Deps: support.Deps{
			Issuer: "http://idp.test", SCL: spec.MustLoadSCL(),
			TenantRepo: tenantRepo,
			Emit:       emit,
		},
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

func TestUpdateBrandingRejectsNonAdmin(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, settingsActor("alice", "acme", nil), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{"product_name": "Acme"})
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func TestUpdateBrandingPersistsAndIsPubliclyVisible(t *testing.T) {
	e, repo, _, events := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := patchBranding(t, e, map[string]any{
		"product_name":  "Acme",
		"primary_color": "#0f172a",
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
	if body.ProductName != "Acme" || body.PrimaryColor != "#0f172a" {
		t.Fatalf("public branding not reflected: %+v", body)
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

func TestUploadAndDeleteBrandingLogoAsset(t *testing.T) {
	e, repo, assetStore, events := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}

	uploadResp := uploadBrandingAsset(t, e, "/realms/acme/api/admin/tenant/branding/assets/logo", png)
	if uploadResp.Code != http.StatusOK {
		t.Fatalf("upload status=%d body=%s", uploadResp.Code, uploadResp.Body.String())
	}
	saved, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || saved == nil || saved.LogoObjectKey == "" {
		t.Fatalf("logo reference not persisted: %+v err=%v", saved, err)
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

	deleteResp := deleteBrandingAsset(t, e, "/realms/acme/api/admin/tenant/branding/assets/logo")
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", deleteResp.Code, deleteResp.Body.String())
	}
	cleared, err := repo.FindByTenant(context.Background(), "acme")
	if err != nil || cleared == nil || cleared.LogoObjectKey != "" {
		t.Fatalf("expected logo cleared: %+v err=%v", cleared, err)
	}
	if len(*events) != 2 {
		t.Fatalf("events=%d want 2 (upload + delete)", len(*events))
	}
}

func TestUploadBrandingAssetRejectsSVG(t *testing.T) {
	e, _, _, _ := newBrandingServer(t, settingsActor("admin", "acme", []string{"admin"}), activeTenant("acme", "Acme"))
	resp := uploadBrandingAsset(t, e, "/realms/acme/api/admin/tenant/branding/assets/logo", []byte("<svg onload=alert(1)></svg>"))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

func patchBranding(t *testing.T, e *echo.Echo, body any) *httptest.ResponseRecorder {
	t.Helper()
	const path = "/realms/acme/api/admin/tenant/branding"
	csrf, cookie := passwordResetContextCSRF(t, e, "/realms/acme/api/auth/password_reset_context")
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

func uploadBrandingAsset(t *testing.T, e *echo.Echo, path string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	csrf, cookie := passwordResetContextCSRF(t, e, tenantPrefix(path)+"/api/auth/password_reset_context")
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
	csrf, cookie := passwordResetContextCSRF(t, e, tenantPrefix(path)+"/api/auth/password_reset_context")
	req := httptest.NewRequest(http.MethodDelete, path, http.NoBody)
	req.Header.Set("Origin", "http://idp.test")
	req.Header.Set("X-Csrf-Token", csrf)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}
