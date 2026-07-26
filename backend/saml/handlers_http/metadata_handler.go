package handlers_http

import (
	"net/http"
	"time"

	metadata "github.com/ambi/idmagic/backend/saml/metadata_saml"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// handleSamlMetadata は realm 単位の SAML 2.0 IdP metadata を公開する。
func (d Deps) handleSamlMetadata(c *echo.Context) error {
	if d.FederationSigner == nil {
		return c.String(http.StatusInternalServerError, "saml metadata unavailable")
	}
	endpoints := metadata.Endpoints{
		SSOURL: support.TenantURL(c, "/saml/sso", d.Issuer),
		SLOURL: support.TenantURL(c, "/saml/slo", d.Issuer),
	}
	now := time.Now().UTC()
	certs, err := d.FederationSigner.Certificates(c.Request().Context(), now)
	if err != nil {
		return c.String(http.StatusInternalServerError, "saml metadata unavailable")
	}
	out, err := metadata.BuildIDPMetadataWithCertificates(support.RequestIssuer(c, d.Issuer), certs, endpoints, now)
	if err != nil {
		return c.String(http.StatusInternalServerError, "saml metadata unavailable")
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Blob(http.StatusOK, "application/xml; charset=utf-8", out)
}
