package handlers_http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/labstack/echo/v5"
)

// wi-285 / scenario Tenancy.tenant_endpoint_style: Subdomain の WebAuthn RP は
// request host から別インスタンスとして導出し、Path は既定の RP をそのまま使う。
func TestResolveRPForRequest(t *testing.T) {
	fallback, err := webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{
		RPID: "idp.example", RPDisplayName: "idmagic", RPOrigins: []string{"https://idp.example"},
	})
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	contextFor := func(style domain.TenantEndpointStyle, issuer, prefix string) *echo.Context {
		req := httptest.NewRequest("GET", "/", http.NoBody)
		req = req.WithContext(tenancy.WithTenant(req.Context(), &domain.Tenant{EndpointStyle: style}, issuer, prefix))
		return e.NewContext(req, httptest.NewRecorder())
	}
	deps := support.Deps{Issuer: "https://idp.example"}
	if got := ResolveRPForRequest(contextFor(domain.TenantEndpointStylePath, "https://idp.example/realms/acme", "/realms/acme"), deps, fallback); got != fallback {
		t.Fatal("path tenant did not retain its configured RP")
	}
	if got := ResolveRPForRequest(contextFor(domain.TenantEndpointStyleSubdomain, "https://acme.idp.example", ""), deps, fallback); got == nil || got == fallback {
		t.Fatal("subdomain tenant did not receive a request-derived RP")
	}
}
