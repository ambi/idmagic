package server_http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

// hostRoutingFixture は default (path) と acme (subdomain) を持つスタックを組む。
func hostRoutingFixture(t *testing.T, baseDomain string) *echo.Echo {
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
			Status: tenancydomain.TenantStatusActive, EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain,
			CreatedAt: now, UpdatedAt: now,
		},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatal(err)
		}
	}
	e := echo.New()
	Register(e, Deps{
		Issuer: "https://idp.example", Contract: spec.CurrentRuntimeContract(),
		TenantRepo: tenants, TenantBaseDomain: baseDomain,
	})
	return e
}

func requestWithHost(host, target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	req.Host = host
	return req
}

// fail-closed: 未登録の Host はどのテナントにも解決してはならない。
// ここが fail-open だと任意の Host ヘッダで default テナントに到達でき、
// テナント境界の破りになる。resolver で最初に固定するのはこの性質。
func TestUnknownSubdomainDoesNotResolveToDefaultTenant(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, requestWithHost("unknown.idp.example", "/.well-known/openid-configuration"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

// prefix 無しの path は default テナントの第 2 ロケーションになるため廃止した。
func TestBareRouteOnBaseDomainIsNotATenantLocation(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, requestWithHost("idp.example", "/.well-known/openid-configuration"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestSubdomainTenantResolvesFromHost(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, requestWithHost("acme.idp.example", "/.well-known/openid-configuration"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Vary"); got != "Host" {
		t.Fatalf("Vary = %q, want Host", got)
	}
	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if got := doc["issuer"]; got != "https://acme.idp.example" {
		t.Fatalf("issuer = %v, want https://acme.idp.example", got)
	}
	if got := doc["authorization_endpoint"]; got != "https://acme.idp.example/authorize" {
		t.Fatalf("authorization_endpoint = %v", got)
	}
}

// 不変条件「1 テナント = 1 正規ロケーション」: テナントは自分の endpoint_style が
// 指す経路からのみ到達でき、他方の経路では不在として扱う。
func TestTenantIsReachableOnlyAtItsCanonicalLocation(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	t.Run("subdomain tenant is absent on the path prefix route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, requestWithHost("idp.example", "/realms/acme/.well-known/openid-configuration"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("path tenant is absent on the subdomain route", func(t *testing.T) {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, requestWithHost("default.idp.example", "/.well-known/openid-configuration"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("a tenant origin cannot reach another tenant through a path prefix", func(t *testing.T) {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, requestWithHost("acme.idp.example", "/realms/default/.well-known/openid-configuration"))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})
}

// base domain 未設定の配備では host 解決そのものが無効で、path 経路だけが残る。
func TestWithoutTenantBaseDomainOnlyPathRoutingApplies(t *testing.T) {
	e := hostRoutingFixture(t, "")

	pathRoute := httptest.NewRecorder()
	e.ServeHTTP(pathRoute, requestWithHost("idp.example", "/realms/default/.well-known/openid-configuration"))
	if pathRoute.Code != http.StatusOK {
		t.Fatalf("path route status = %d, body = %s", pathRoute.Code, pathRoute.Body.String())
	}
	var doc map[string]any
	if err := json.Unmarshal(pathRoute.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if got := doc["issuer"]; got != "https://idp.example/realms/default" {
		t.Fatalf("issuer = %v", got)
	}

	hostRoute := httptest.NewRecorder()
	e.ServeHTTP(hostRoute, requestWithHost("acme.idp.example", "/.well-known/openid-configuration"))
	if hostRoute.Code != http.StatusNotFound {
		t.Fatalf("host route status = %d, body = %s", hostRoute.Code, hostRoute.Body.String())
	}
}

// Host は port と trailing dot を除き lowercase に正規化して照合する (SCL ResolveTenant)。
func TestHostIsNormalizedBeforeMatching(t *testing.T) {
	e := hostRoutingFixture(t, "idp.example")

	for _, host := range []string{"ACME.idp.example", "acme.idp.example:8443", "acme.idp.example."} {
		t.Run(host, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, requestWithHost(host, "/.well-known/openid-configuration"))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
			}
		})
	}
}
