package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

func newTenantMiddlewareContext(target string) *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestTenantRouteCookiePathAndName(t *testing.T) {
	pathTenant := &tenancydomain.Tenant{ID: "t1", Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStylePath}
	subTenant := &tenancydomain.Tenant{ID: "t2", Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain}

	t.Run("path style tenant", func(t *testing.T) {
		c := newTenantMiddlewareContext("http://idp.test/anything")
		c.SetRequest(c.Request().WithContext(tenantports.WithTenant(c.Request().Context(), pathTenant, "https://idp.test/realms/acme", "/realms/acme")))
		if got := TenantRoute(c, "/foo"); got != "/realms/acme/foo" {
			t.Fatalf("TenantRoute = %q", got)
		}
		if got := TenantCookiePath(c); got != "/realms/acme" {
			t.Fatalf("TenantCookiePath = %q", got)
		}
		if got := TenantCookieName(c, CSRFCookie); got != CSRFCookie {
			t.Fatalf("TenantCookieName = %q, want unprefixed", got)
		}
		if TenantCookieSecure(c) {
			t.Fatal("path style tenant must not require Secure __Host- cookies")
		}
	})

	t.Run("subdomain style tenant", func(t *testing.T) {
		c := newTenantMiddlewareContext("http://acme.idp.test/anything")
		c.SetRequest(c.Request().WithContext(tenantports.WithTenant(c.Request().Context(), subTenant, "https://acme.idp.test", "")))
		if got := TenantRoute(c, "/foo"); got != "/foo" {
			t.Fatalf("TenantRoute = %q", got)
		}
		if got := TenantCookiePath(c); got != "/" {
			t.Fatalf("TenantCookiePath = %q", got)
		}
		if got := TenantCookieName(c, CSRFCookie); got != "__Host-"+CSRFCookie {
			t.Fatalf("TenantCookieName = %q, want __Host- prefixed", got)
		}
		if !TenantCookieSecure(c) {
			t.Fatal("subdomain style tenant must require Secure __Host- cookies")
		}
	})

	t.Run("no tenant in context falls back to root", func(t *testing.T) {
		c := newTenantMiddlewareContext("http://idp.test/anything")
		if got := TenantRoute(c, "/foo"); got != "/foo" {
			t.Fatalf("TenantRoute = %q", got)
		}
		if got := TenantCookiePath(c); got != "/" {
			t.Fatalf("TenantCookiePath = %q", got)
		}
		if got := TenantCookieName(c, CSRFCookie); got != CSRFCookie {
			t.Fatalf("TenantCookieName = %q", got)
		}
		if TenantCookieSecure(c) {
			t.Fatal("no tenant in context must not require Secure cookies")
		}
	})
}

// htu の期待値は、クライアントがリクエストを送った絶対 URL である。path style の issuer は
// すでに /realms/{realm} を含むので、issuer 全体にパスを継ぐと prefix が二重になる。基底の
// issuer を前置すると subdomain style の host が落ちる。どちらの誤りもここで区別する。
//
//spec:covers RFC9449-API-TOKEN-DPOP: htu の期待値を、テナントの正規ロケーションの origin とリクエストのパスから、クエリを除いて組む。
func TestRequestHTUIsTheTargetURIAtTheCanonicalLocation(t *testing.T) {
	pathTenant := &tenancydomain.Tenant{ID: "acme", Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStylePath}
	subTenant := &tenancydomain.Tenant{ID: "globex", Realm: "globex", EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain}
	for _, tc := range []struct {
		name, target     string
		tenant           *tenancydomain.Tenant
		issuer, fallback string
		want             string
	}{
		{
			name: "path style", target: "http://internal:8080/realms/acme/api/admin/v1/users?limit=1",
			tenant: pathTenant, issuer: "https://idp.test/realms/acme",
			want: "https://idp.test/realms/acme/api/admin/v1/users",
		},
		{
			name: "subdomain style", target: "http://internal:8080/token",
			tenant: subTenant, issuer: "https://globex.idp.test:8443", fallback: "https://idp.test",
			want: "https://globex.idp.test:8443/token",
		},
		{
			name: "no tenant falls back", target: "http://internal/realms/default/token",
			fallback: "https://idp.test/",
			want:     "https://idp.test/realms/default/token",
		},
		{name: "no issuer at all", target: "http://internal/realms/default/token", want: ""},
		{name: "issuer without a host", target: "http://internal/token", fallback: "idp.test", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTenantMiddlewareContext(tc.target)
			if tc.tenant != nil {
				c.SetRequest(c.Request().WithContext(tenantports.WithTenant(c.Request().Context(), tc.tenant, tc.issuer, "")))
			}
			if got := RequestHTU(c, tc.fallback); got != tc.want {
				t.Fatalf("RequestHTU = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTenantURLFallsBackWithoutATenant(t *testing.T) {
	c := newTenantMiddlewareContext("http://idp.test/realms/acme/token?foo=bar")
	if got := TenantURL(c, "/x", "https://fallback.test"); got != "https://fallback.test/x" {
		t.Fatalf("TenantURL = %q", got)
	}
}

func TestTenantIDOnlyContextDropsCookieScope(t *testing.T) {
	c := newTenantMiddlewareContext("https://http.example/account")
	ctx := tenantports.WithTenant(c.Request().Context(), &tenancydomain.Tenant{ID: "http", EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain}, "https://http.example", "/realms/http")
	c.SetRequest(c.Request().WithContext(tenantports.WithTenantID(ctx, "job")))
	if got := RequestTenantID(c); got != "job" {
		t.Fatalf("RequestTenantID() = %q, want job", got)
	}
	if got := TenantCookieName(c, "session"); got != "session" {
		t.Fatalf("TenantCookieName() = %q, want session", got)
	}
	if got := TenantCookiePath(c); got != "/" {
		t.Fatalf("TenantCookiePath() = %q, want /", got)
	}
	if TenantCookieSecure(c) {
		t.Fatal("ID-only context retained host-only cookie attributes")
	}
}
