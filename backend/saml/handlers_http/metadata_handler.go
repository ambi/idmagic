package handlers_http

import (
	"encoding/pem"
	"net/http"
	"time"

	metadata "github.com/ambi/idmagic/backend/saml/metadata_saml"

	"github.com/labstack/echo/v5"
)

// handleSamlMetadata は realm 単位の SAML 2.0 IdP metadata を公開する。
func (d Deps) handleSamlMetadata(c *echo.Context) error {
	if d.FederationSigner == nil {
		return c.String(http.StatusInternalServerError, "saml metadata unavailable")
	}
	profile, err := d.requireIDPProfile(c)
	if err != nil {
		return err
	}
	endpoints := metadata.Endpoints{
		SSOURL: d.idpProfileURL(c, profile.ProfileID, "sso"),
		SLOURL: d.idpProfileURL(c, profile.ProfileID, "slo"),
	}
	now := time.Now().UTC()
	certs, err := d.FederationSigner.Certificates(idpProfileSigningContext(c, profile.ProfileID), now)
	if err != nil {
		return c.String(http.StatusInternalServerError, "saml metadata unavailable")
	}
	out, err := metadata.BuildIDPMetadataWithCertificates(d.idpProfileEntityID(c, profile.ProfileID), certs, endpoints, now)
	if err != nil {
		return c.String(http.StatusInternalServerError, "saml metadata unavailable")
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Blob(http.StatusOK, "application/xml; charset=utf-8", out)
}

// handleSamlSigningCertificate は新規 SP 設定に使う active XML federation 証明書を
// PEM で返す。rotation overlap の全証明書は SAML metadata を正本とする。
func (d Deps) handleSamlSigningCertificate(c *echo.Context) error {
	if d.FederationSigner == nil {
		return c.String(http.StatusInternalServerError, "saml signing certificate unavailable")
	}
	profile, err := d.requireIDPProfile(c)
	if err != nil {
		return err
	}
	signer, err := d.FederationSigner.Resolve(idpProfileSigningContext(c, profile.ProfileID))
	if err != nil || signer == nil || signer.Certificate() == nil {
		return c.String(http.StatusInternalServerError, "saml signing certificate unavailable")
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: signer.Certificate().Raw})
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set("Content-Disposition", `attachment; filename="idmagic-saml-signing-certificate.pem"`)
	return c.Blob(http.StatusOK, "application/x-pem-file; charset=utf-8", pemBytes)
}
