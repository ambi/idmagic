package handlers_http

import (
	"net/http"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	metadata "github.com/ambi/idmagic/backend/wsfederation/metadata_wsfederation"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleFederationMetadata(c *echo.Context) error {
	endpoints := d.federationEndpoints(c)
	out, err := metadata.BuildFederationMetadata(support.RequestIssuer(c, d.Issuer), d.FederationSigner.Certificate(), endpoints, time.Now().UTC())
	if err != nil {
		return c.String(http.StatusInternalServerError, "federation metadata unavailable")
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Blob(http.StatusOK, "application/xml; charset=utf-8", out)
}

func (d Deps) handleTrustMEX(c *echo.Context) error {
	out, err := metadata.BuildMEX(d.federationEndpoints(c))
	if err != nil {
		return c.String(http.StatusInternalServerError, "trust metadata unavailable")
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Blob(http.StatusOK, "application/xml; charset=utf-8", out)
}

func (d Deps) federationEndpoints(c *echo.Context) metadata.EndpointSet {
	return metadata.EndpointSet{
		PassiveURL:        support.TenantURL(c, "/wsfed", d.Issuer),
		ActiveURL:         support.TenantURL(c, "/trust/usernamemixed", d.Issuer),
		MEXURL:            support.TenantURL(c, "/trust/mex", d.Issuer),
		FederationMetaURL: support.TenantURL(c, "/federationmetadata/2007-06/federationmetadata.xml", d.Issuer),
	}
}
