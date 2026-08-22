package support_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	tenancy "github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func TestNormalizeHost(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Acme.Example.COM", "acme.example.com"},
		{"acme.example.com:443", "acme.example.com"},
		{"  acme.example.com. ", "acme.example.com"},
		{"[::1]:8080", "::1"},
		{"[::1]", "::1"},
	}
	for _, tc := range cases {
		if got := normalizeHost(tc.in); got != tc.want {
			t.Errorf("normalizeHost(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestHostRealm(t *testing.T) {
	d := Deps{TenantBaseDomain: "idmagic.test"}

	t.Run("empty base domain disables host resolution", func(t *testing.T) {
		empty := Deps{}
		if _, ok := empty.hostRealm("acme.idmagic.test"); ok {
			t.Fatal("expected host resolution disabled with empty TenantBaseDomain")
		}
	})

	t.Run("matches a single-label realm", func(t *testing.T) {
		realm, ok := d.hostRealm("acme.idmagic.test:443")
		if !ok || realm != "acme" {
			t.Fatalf("hostRealm = %q, %v", realm, ok)
		}
	})

	t.Run("rejects a host outside the base domain", func(t *testing.T) {
		if _, ok := d.hostRealm("acme.other.test"); ok {
			t.Fatal("expected no match for a foreign base domain")
		}
	})

	t.Run("rejects the bare base domain with no label", func(t *testing.T) {
		if _, ok := d.hostRealm("idmagic.test"); ok {
			t.Fatal("expected no match for the bare base domain")
		}
	})

	t.Run("rejects a multi-label host to prevent realm spoofing", func(t *testing.T) {
		if _, ok := d.hostRealm("evil.acme.idmagic.test"); ok {
			t.Fatal("expected multi-label host to be rejected")
		}
	})
}

func newTenantMiddlewareContext(target string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestResolveHostTenantDenyByDefault(t *testing.T) {
	d := Deps{TenantBaseDomain: "idmagic.test"}
	called := false
	next := func(*echo.Context) error { called = true; return nil }

	c, rec := newTenantMiddlewareContext("http://unrelated.example/anything")
	if err := d.ResolveHostTenant(next)(c); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("next must not run when the host does not resolve to a tenant")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestResolveHostTenantEntersSubdomainTenant(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	tenant := &tenancydomain.Tenant{
		ID: "tenant-acme", Realm: "acme", Status: tenancydomain.TenantStatusActive,
		EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain,
	}
	if err := tenants.Save(t.Context(), tenant); err != nil {
		t.Fatal(err)
	}
	d := Deps{TenantBaseDomain: "idmagic.test", TenantRepo: tenants, Issuer: "https://idmagic.test"}

	var seenTenantID string
	next := func(c *echo.Context) error {
		seenTenantID = RequestTenantID(c)
		return nil
	}
	c, rec := newTenantMiddlewareContext("http://acme.idmagic.test/anything")
	if err := d.ResolveHostTenant(next)(c); err != nil {
		t.Fatal(err)
	}
	if seenTenantID != "tenant-acme" {
		t.Fatalf("seenTenantID=%q", seenTenantID)
	}
	if rec.Header().Get("Vary") != "Host" {
		t.Fatalf("Vary header = %q, want Host", rec.Header().Get("Vary"))
	}
}

func TestResolveHostTenantRejectsPathStyleTenant(t *testing.T) {
	// A tenant registered as path-style must not resolve through the host route,
	// even if its realm happens to match a subdomain label.
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(t.Context(), &tenancydomain.Tenant{
		ID: "tenant-acme", Realm: "acme", Status: tenancydomain.TenantStatusActive,
		EndpointStyle: tenancydomain.TenantEndpointStylePath,
	}); err != nil {
		t.Fatal(err)
	}
	d := Deps{TenantBaseDomain: "idmagic.test", TenantRepo: tenants}
	c, rec := newTenantMiddlewareContext("http://acme.idmagic.test/anything")
	if err := d.ResolveHostTenant(func(*echo.Context) error { return nil })(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404 for a path-style tenant reached via host", rec.Code)
	}
}

func TestResolveHostTenantRejectsDisabledTenant(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(t.Context(), &tenancydomain.Tenant{
		ID: "tenant-acme", Realm: "acme", Status: tenancydomain.TenantStatusDisabled,
		EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain,
	}); err != nil {
		t.Fatal(err)
	}
	d := Deps{TenantBaseDomain: "idmagic.test", TenantRepo: tenants}
	c, rec := newTenantMiddlewareContext("http://acme.idmagic.test/anything")
	if err := d.ResolveHostTenant(func(*echo.Context) error { return nil })(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400 for a disabled tenant", rec.Code)
	}
}

func TestResolvePathTenantRejectsHostResolvableOrigin(t *testing.T) {
	// A subdomain tenant's own origin must not also accept a path-prefixed
	// escape into a different tenant (cross-tenant cookie boundary).
	d := Deps{TenantBaseDomain: "idmagic.test"}
	c, rec := newTenantMiddlewareContext("http://acme.idmagic.test/realms/other/anything")
	c.SetPathValues(echo.PathValues{{Name: "tenant_id", Value: "other"}})
	if err := d.ResolvePathTenant(func(*echo.Context) error { return nil })(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

func TestResolvePathTenantEntersPathStyleTenant(t *testing.T) {
	tenants := tenancymemory.NewTenantRepository()
	if err := tenants.Save(t.Context(), &tenancydomain.Tenant{
		ID: "tenant-acme", Realm: "acme", Status: tenancydomain.TenantStatusActive,
		EndpointStyle: tenancydomain.TenantEndpointStylePath,
	}); err != nil {
		t.Fatal(err)
	}
	d := Deps{TenantRepo: tenants, Issuer: "https://idp.test"}
	c, _ := newTenantMiddlewareContext("http://idp.test/realms/acme/anything")
	c.SetPathValues(echo.PathValues{{Name: "tenant_id", Value: "acme"}})
	var seenTenantID string
	next := func(c *echo.Context) error { seenTenantID = RequestTenantID(c); return nil }
	if err := d.ResolvePathTenant(next)(c); err != nil {
		t.Fatal(err)
	}
	if seenTenantID != "tenant-acme" {
		t.Fatalf("seenTenantID=%q", seenTenantID)
	}
}

func TestResolveDefaultRealmTenantWithoutRepoUsesBuiltinDefault(t *testing.T) {
	d := Deps{Issuer: "https://idp.test"}
	c, _ := newTenantMiddlewareContext("http://idp.test/realms/default/anything")
	var seenTenantID string
	next := func(c *echo.Context) error { seenTenantID = RequestTenantID(c); return nil }
	if err := d.ResolveDefaultRealmTenant(next)(c); err != nil {
		t.Fatal(err)
	}
	if seenTenantID != tenancydomain.DefaultTenantID {
		t.Fatalf("seenTenantID=%q, want default tenant", seenTenantID)
	}
}

func TestResolveDefaultRealmTenantRejectsHostResolvableOrigin(t *testing.T) {
	d := Deps{TenantBaseDomain: "idmagic.test"}
	c, rec := newTenantMiddlewareContext("http://acme.idmagic.test/realms/default/anything")
	if err := d.ResolveDefaultRealmTenant(func(*echo.Context) error { return nil })(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rec.Code)
	}
}

func TestResolveTenantWithNilRepoRejectsNonDefaultRealm(t *testing.T) {
	d := Deps{}
	c, rec := newTenantMiddlewareContext("http://idp.test/realms/acme/anything")
	c.SetPathValues(echo.PathValues{{Name: "tenant_id", Value: "acme"}})
	if err := d.ResolvePathTenant(func(*echo.Context) error { return nil })(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404 for a non-default realm with no TenantRepo", rec.Code)
	}
}

func TestResolveTenantPropagatesRepoError(t *testing.T) {
	d := Deps{TenantRepo: erroringTenantRepo{}}
	c, _ := newTenantMiddlewareContext("http://idp.test/realms/acme/anything")
	c.SetPathValues(echo.PathValues{{Name: "tenant_id", Value: "acme"}})
	err := d.ResolvePathTenant(func(*echo.Context) error { return nil })(c)
	if err == nil {
		t.Fatal("expected the repository error to propagate")
	}
}

func TestCanonicalLocationPathStyle(t *testing.T) {
	d := Deps{Issuer: "https://idp.test"}
	tenant := &tenancydomain.Tenant{Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStylePath}
	issuer, prefix := d.CanonicalLocation(tenant)
	if issuer != "https://idp.test/realms/acme" || prefix != "/realms/acme" {
		t.Fatalf("issuer=%q prefix=%q", issuer, prefix)
	}
}

func TestCanonicalLocationSubdomainStyle(t *testing.T) {
	cases := []struct {
		name       string
		issuer     string
		wantIssuer string
	}{
		{"https issuer", "https://idmagic.test", "https://acme.idmagic.test"},
		{"http issuer preserves scheme and port", "http://localhost:5173", "http://acme.idmagic.test:5173"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Deps{Issuer: tc.issuer, TenantBaseDomain: "idmagic.test"}
			tenant := &tenancydomain.Tenant{Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain}
			issuer, prefix := d.CanonicalLocation(tenant)
			if issuer != tc.wantIssuer || prefix != "" {
				t.Fatalf("issuer=%q prefix=%q, want issuer=%q prefix=empty", issuer, prefix, tc.wantIssuer)
			}
		})
	}
}

func TestTenantRouteCookiePathAndName(t *testing.T) {
	pathTenant := &tenancydomain.Tenant{ID: "t1", Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStylePath}
	subTenant := &tenancydomain.Tenant{ID: "t2", Realm: "acme", EndpointStyle: tenancydomain.TenantEndpointStyleSubdomain}

	t.Run("path style tenant", func(t *testing.T) {
		c, _ := newTenantMiddlewareContext("http://idp.test/anything")
		c.SetRequest(c.Request().WithContext(tenancy.WithTenant(c.Request().Context(), pathTenant, "https://idp.test/realms/acme", "/realms/acme")))
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
		c, _ := newTenantMiddlewareContext("http://acme.idp.test/anything")
		c.SetRequest(c.Request().WithContext(tenancy.WithTenant(c.Request().Context(), subTenant, "https://acme.idp.test", "")))
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
		c, _ := newTenantMiddlewareContext("http://idp.test/anything")
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

func TestRequestHTUAndTenantURL(t *testing.T) {
	c, _ := newTenantMiddlewareContext("http://idp.test/realms/acme/token?foo=bar")
	if got := RequestHTU(c, "https://idp.test/realms/acme"); got != "https://idp.test/realms/acme/realms/acme/token" {
		t.Fatalf("RequestHTU = %q", got)
	}
	if got := TenantURL(c, "/x", "https://fallback.test"); got != "https://fallback.test/x" {
		t.Fatalf("TenantURL = %q", got)
	}
}

// erroringTenantRepo implements tenantports.TenantRepository with every
// method failing, to exercise resolveTenant's repository-error passthrough.
type erroringTenantRepo struct{}

func (erroringTenantRepo) FindByID(context.Context, string) (*tenancydomain.Tenant, error) {
	return nil, errors.New("boom")
}

func (erroringTenantRepo) FindByRealm(context.Context, string) (*tenancydomain.Tenant, error) {
	return nil, errors.New("boom")
}

func (erroringTenantRepo) FindAll(context.Context) ([]*tenancydomain.Tenant, error) {
	return nil, errors.New("boom")
}

func (erroringTenantRepo) Save(context.Context, *tenancydomain.Tenant) error {
	return errors.New("boom")
}
