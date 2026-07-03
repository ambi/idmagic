package http_test

// SCL scenario "管理者は自テナントの署名鍵を参照し、admin / system_admin が
// 自テナントの鍵をローテートできる (per-tenant)" を /api/admin/keys 経由で検証する。
// - AdminKeysRead: admin / system_admin どちらでも自テナントの List/Get 可能
// - TenantKeysRotate: admin / system_admin が自テナントに対して実行できる

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdomain "idmagic/internal/authentication/domain"
	oauth2http "idmagic/internal/oauth2/adapters/http"
	"idmagic/internal/shared/adapters/crypto"
	httpadapter "idmagic/internal/shared/adapters/http/server"
	"idmagic/internal/shared/adapters/http/support"
	"idmagic/internal/shared/adapters/persistence/memory"
	"idmagic/internal/shared/spec"
	"idmagic/internal/tenancy"

	"github.com/labstack/echo/v5"
)

func newKeyAdminServer(t *testing.T, actor *spec.User) (*echo.Echo, *crypto.InMemoryKeyStore, *[]spec.DomainEvent) {
	t.Helper()
	userRepo := memory.NewUserRepository()
	if actor != nil {
		userRepo.Seed(actor)
	}
	keyStore, err := crypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	resolver := &fakeAuthnResolver{}
	if actor != nil {
		resolver.ctx = &authdomain.AuthenticationContext{
			Sub: actor.Sub, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
		}
	}
	events := make([]spec.DomainEvent, 0)
	emit := func(e spec.DomainEvent) { events = append(events, e) }
	e := echo.New()
	httpadapter.Register(e, support.Deps{
		Issuer: "http://idp.test", SCL: spec.MustLoadSCL(), UserRepo: userRepo,
		KeyStore: keyStore, AuthnResolver: resolver,
		TenantRepo: newSingleTenantRepo(),
		Emit:       emit,
	})
	return e, keyStore, &events
}

func keyAdminUser(sub, tenantID string, roles []string) *spec.User {
	now := time.Now().UTC()
	return &spec.User{
		Sub: sub, PreferredUsername: sub, TenantID: tenantID, Roles: roles,
		CreatedAt: now, UpdatedAt: now,
	}
}

func getAdminKeys(e *echo.Echo, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func postRotate(t *testing.T, e *echo.Echo, path string) *httptest.ResponseRecorder {
	t.Helper()
	// CSRF token / cookie は password_reset_context 経由で発行する。
	csrf, cookie := passwordResetContextCSRF(t, e, "/realms/default/api/auth/password_reset_context")
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte("{}")))
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
		t.Fatalf("csrf bootstrap status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		CSRFToken string `json:"csrf_token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	result := rec.Result()
	defer result.Body.Close()
	cookies := result.Cookies()
	if len(cookies) == 0 || body.CSRFToken == "" {
		t.Fatalf("csrf=%q cookies=%v", body.CSRFToken, cookies)
	}
	return body.CSRFToken, cookies[0]
}

func TestAdminKeysListRequiresAdminRole(t *testing.T) {
	plain := keyAdminUser("user_alice", "acme", []string{})
	e, _, _ := newKeyAdminServer(t, plain)
	rec := getAdminKeys(e, "/realms/acme/api/admin/keys")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminKeysListReturnsAllKeys(t *testing.T) {
	user := keyAdminUser("user_admin", "acme", []string{"admin"})
	e, keyStore, _ := newKeyAdminServer(t, user)
	// acme テナントの鍵を 2 本作り JWKS 上に active+verifying を作る。
	// KeyStore は tenant-aware なので acme の ctx で回転する。
	acmeCtx := tenancy.WithTenant(context.Background(), &spec.Tenant{ID: "acme"}, "", "")
	if _, err := keyStore.Rotate(acmeCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := keyStore.Rotate(acmeCtx); err != nil {
		t.Fatal(err)
	}
	rec := getAdminKeys(e, "/realms/acme/api/admin/keys")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Keys []oauth2http.AdminKeyResponse `json:"keys"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Keys) != 2 {
		t.Fatalf("keys=%d want 2", len(body.Keys))
	}
	active := 0
	for _, k := range body.Keys {
		if k.Active {
			active++
		}
		if _, ok := k.PublicJWK["n"]; !ok {
			t.Fatalf("public JWK must include RSA modulus n: %+v", k.PublicJWK)
		}
	}
	if active != 1 {
		t.Fatalf("exactly one active key expected, got %d", active)
	}
}

func TestAdminKeysGetUnknownKidReturns404(t *testing.T) {
	user := keyAdminUser("user_admin", "acme", []string{"admin"})
	e, _, _ := newKeyAdminServer(t, user)
	rec := getAdminKeys(e, "/realms/acme/api/admin/keys/unknown-kid")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminKeysRotateAllowsTenantAdmin(t *testing.T) {
	// per-tenant 鍵のため、admin は自テナントの鍵を回転できる。
	admin := keyAdminUser("user_admin", spec.DefaultTenantID, []string{"admin"})
	e, _, _ := newKeyAdminServer(t, admin)
	rec := postRotate(t, e, "/realms/default/api/admin/keys/rotate")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminKeysRotateRejectsNonAdmin(t *testing.T) {
	// admin / system_admin いずれのロールも持たないユーザーは回転できない。
	plain := keyAdminUser("user_alice", spec.DefaultTenantID, []string{})
	e, _, _ := newKeyAdminServer(t, plain)
	rec := postRotate(t, e, "/realms/default/api/admin/keys/rotate")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminKeysRotateRejectsSystemAdminOutsideDefaultPath(t *testing.T) {
	// system_admin (TenantID=default) が /realms/acme/.../rotate に到達した場合、
	// 二段の防御で reject される:
	//   1. resolveAuthentication が user.TenantID != support.RequestTenantID(=acme) で
	//      セッションを未認証扱いし 401 を返す (defense-in-depth)
	//   2. もし 1 を抜けても requireTenantKeyManager が actor.TenantID != request tenant で 403
	// 期待される挙動は (1) が先に発火するため 401。
	sysAdmin := keyAdminUser("user_sys", spec.DefaultTenantID, []string{"system_admin"})
	e, _, _ := newKeyAdminServer(t, sysAdmin)
	rec := postRotate(t, e, "/realms/acme/api/admin/keys/rotate")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminKeysRotateSucceedsAndEmitsEvent(t *testing.T) {
	sysAdmin := keyAdminUser("user_sys", spec.DefaultTenantID, []string{"system_admin"})
	e, keyStore, events := newKeyAdminServer(t, sysAdmin)
	prevActive, err := keyStore.GetActiveKey(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rec := postRotate(t, e, "/realms/default/api/admin/keys/rotate")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body oauth2http.AdminRotateKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.Next.Active {
		t.Fatalf("next key must be active: %+v", body.Next)
	}
	if body.Previous == nil || body.Previous.Kid != prevActive.Kid {
		t.Fatalf("previous kid mismatch: prev=%+v want=%s", body.Previous, prevActive.Kid)
	}
	if body.Previous.Active {
		t.Fatalf("previous key must be non-active after rotation: %+v", body.Previous)
	}
	if len(*events) != 1 {
		t.Fatalf("expected 1 emitted event, got %d", len(*events))
	}
	rotated, ok := (*events)[0].(*spec.SigningKeyRotated)
	if !ok {
		t.Fatalf("event type=%T, want *spec.SigningKeyRotated", (*events)[0])
	}
	// wi-36 から繰り延べた残課題: 回転イベントは帰属テナントを持つ。
	if rotated.TenantID != spec.DefaultTenantID {
		t.Fatalf("SigningKeyRotated.TenantID=%q want %q", rotated.TenantID, spec.DefaultTenantID)
	}
}

func TestAdminKeysHealthListsPerTenantHealth(t *testing.T) {
	sysAdmin := keyAdminUser("user_sys", spec.DefaultTenantID, []string{"system_admin"})
	e, _, _ := newKeyAdminServer(t, sysAdmin)
	rec := getAdminKeys(e, "/realms/default/api/admin/keys/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Tenants []oauth2http.TenantKeyHealthResponse `json:"tenants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Tenants) == 0 {
		t.Fatal("health must list at least the default tenant")
	}
	for _, h := range body.Tenants {
		if h.Provider == "" || h.Usage != string(spec.KeyUsageSigning) {
			t.Fatalf("unexpected health entry: %+v", h)
		}
	}
}

func TestAdminKeysHealthRejectsPlainAdmin(t *testing.T) {
	admin := keyAdminUser("user_admin", spec.DefaultTenantID, []string{"admin"})
	e, _, _ := newKeyAdminServer(t, admin)
	rec := getAdminKeys(e, "/realms/default/api/admin/keys/health")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
