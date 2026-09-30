package server_http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// disabledTenantFixture は default (path)、無効化した acme (subdomain)、無効化した beta (path) を持つ。
// 無効化の応答が到達経路でどう変わるかを読むには、両方の形の無効なテナントが要る。
func disabledTenantFixture(t *testing.T) *echo.Echo {
	t.Helper()
	tenants := tenancymemory.NewTenantRepository()
	now := time.Now().UTC()
	for _, tenant := range []*tenancydomain.Tenant{
		{
			ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default",
			Status: tenancydomain.TenantStatusActive, EndpointStyle: tenancydomain.TenantEndpointStylePath,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "11111111-1111-4111-8111-111111111111", Realm: "acme", DisplayName: "Acme",
			Status: tenancydomain.TenantStatusDisabled, EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain,
			CreatedAt: now, UpdatedAt: now, DisabledAt: &now,
		},
		{
			ID: "22222222-2222-4222-8222-222222222222", Realm: "beta", DisplayName: "Beta",
			Status: tenancydomain.TenantStatusDisabled, EndpointStyle: tenancydomain.TenantEndpointStylePath,
			CreatedAt: now, UpdatedAt: now, DisabledAt: &now,
		},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	e := echo.New()
	Register(e, Deps{
		Issuer: "https://idp.example", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, TenantBaseDomain: "idp.example",
	})
	return e
}

func serve(e *echo.Echo, host, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, requestWithHost(host, target))
	return rec
}

func assertTenantNotFound(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s, want 404", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body = %s: %v", rec.Body.String(), err)
	}
	if len(body) != 1 || body["error"] != "tenant_not_found" {
		t.Fatalf("body = %s, want exactly {\"error\":\"tenant_not_found\"}", rec.Body.String())
	}
}

// realm の左にラベルを重ねた Host は realm ではない。ここで弾かないと
// `evil.acme.{base}` という任意のホスト名で acme に到達できる。
//
//spec:covers EX-TENANCY-022-02: 二段のラベルを持つ Host は、右のラベルが実在の realm でも解決されず 404 tenant_not_found になる。
func TestMultiLabelHostDoesNotResolveToTheInnerRealm(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	// 対照: 一段のラベルなら同じ realm に解決する。
	if own := serve(e, "acme.idp.example", "/.well-known/openid-configuration"); own.Code != http.StatusOK {
		t.Fatalf("control status = %d, body = %s", own.Code, own.Body.String())
	}
	assertTenantNotFound(t, serve(e, "evil.acme.idp.example", "/.well-known/openid-configuration"))
}

// 無効化されたテナントへ正規ロケーション以外から来た要求は、存在しない realm と同じ応答にする。
// 400 を返すと、到達経路の外からテナントの存在と状態が読める。
//
//spec:covers EX-TENANCY-023-01: 無効な subdomain テナントへ path で到達すると、存在しない realm と同じ 404 本文になる。
func TestDisabledTenantOffItsCanonicalLocationLooksAbsent(t *testing.T) {
	e := disabledTenantFixture(t)

	disabled := serve(e, "idp.example", "/realms/acme/.well-known/openid-configuration")
	missing := serve(e, "idp.example", "/realms/ghost/.well-known/openid-configuration")
	assertTenantNotFound(t, disabled)
	if disabled.Body.String() != missing.Body.String() {
		t.Fatalf("disabled = %s, missing = %s, want identical", disabled.Body.String(), missing.Body.String())
	}
}

// 無効化の拒否はプロトコルの経路に限らない。テナントを経由するすべての経路で同じ 400 になる。
//
//spec:covers EX-TENANCY-023-02: 無効な path テナントの公開ブランド設定の経路は 400 invalid_request を返し、ブランド設定を返さない。
func TestDisabledTenantRefusesNonProtocolRoutes(t *testing.T) {
	e := disabledTenantFixture(t)

	for _, target := range []string{"/realms/beta/api/branding", "/realms/beta/.well-known/openid-configuration"} {
		rec := serve(e, "idp.example", target)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, body = %s, want 400", target, rec.Code, rec.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["error"] != "invalid_request" {
			t.Fatalf("%s: body = %s", target, rec.Body.String())
		}
		if _, ok := body["product_name"]; ok {
			t.Fatalf("%s: body carries branding: %s", target, rec.Body.String())
		}
	}
}

// 正規ロケーションは Host に依存する。共有キャッシュが Host をキーにせずに discovery を
// 混ぜると、別テナントの発行者が返る。
//
//spec:covers EX-TENANCY-024-01: 解決に成功した応答は Vary: Host を持つ。
func TestResolvedTenantResponsesVaryByHost(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	for _, request := range []struct{ host, target string }{
		{"idp.example", "/realms/default/.well-known/openid-configuration"},
		{"acme.idp.example", "/.well-known/openid-configuration"},
	} {
		rec := serve(e, request.host, request.target)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s%s: status = %d", request.host, request.target, rec.Code)
		}
		if got := rec.Header().Values("Vary"); !slices.Contains(got, "Host") {
			t.Fatalf("%s%s: Vary = %v, want Host", request.host, request.target, got)
		}
	}
}
