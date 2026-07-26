package support_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// wi-285 / scenario Tenancy.tenant_endpoint_style: subdomain の cookie は
// __Host- prefix と root path を持ち、path style は従来の名前と realm prefix を保つ。
func TestTenantCookieScope(t *testing.T) {
	e := echo.New()
	for _, tc := range []struct {
		name, prefix, wantName, wantPath string
		wantSecure                       bool
		style                            domain.TenantEndpointStyle
	}{
		{"path", "/realms/acme", "idmagic_session", "/realms/acme", false, domain.TenantEndpointStylePath},
		{"subdomain", "", "__Host-idmagic_session", "/", true, domain.TenantEndpointStyleSubdomain},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", http.NoBody)
			req = req.WithContext(tenancy.WithTenant(req.Context(), &domain.Tenant{EndpointStyle: tc.style}, "", tc.prefix))
			c := e.NewContext(req, httptest.NewRecorder())
			if got := TenantCookieName(c, "idmagic_session"); got != tc.wantName {
				t.Fatalf("cookie name = %q, want %q", got, tc.wantName)
			}
			if got := TenantCookiePath(c); got != tc.wantPath {
				t.Fatalf("cookie path = %q, want %q", got, tc.wantPath)
			}
			if got := TenantCookieSecure(c); got != tc.wantSecure {
				t.Fatalf("cookie secure = %v, want %v", got, tc.wantSecure)
			}
		})
	}
}
