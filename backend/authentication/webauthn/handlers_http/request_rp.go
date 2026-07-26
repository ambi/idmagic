package handlers_http

import (
	"net/url"

	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/labstack/echo/v5"
)

// ResolveRPForRequest derives a subdomain tenant's WebAuthn RP from its
// canonical request location. Path tenants keep the process-wide RP supplied
// at startup. The fallback is also the feature-enable switch: without an
// explicitly configured WebAuthn RP, no dynamic RP is constructed.
func ResolveRPForRequest(c *echo.Context, deps support.Deps, fallback *gowebauthn.WebAuthn) *gowebauthn.WebAuthn {
	if fallback == nil || !support.TenantCookieSecure(c) {
		return fallback
	}
	issuer := support.RequestIssuer(c, deps.Issuer)
	parsed, err := url.Parse(issuer)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil
	}
	host := parsed.Hostname()
	if host == "" {
		return nil
	}
	origin := parsed.Scheme + "://" + parsed.Host
	// Hostname intentionally excludes the port: WebAuthn RP IDs are DNS names,
	// while the origin retains a development port when one is present.
	rp, err := webauthnusecases.NewWebAuthn(webauthnusecases.WebAuthnConfig{
		RPID: host, RPDisplayName: "idmagic", RPOrigins: []string{origin},
	})
	if err != nil {
		return nil
	}
	return rp
}
