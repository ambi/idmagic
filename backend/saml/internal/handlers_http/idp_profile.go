package handlers_http

import (
	"context"
	"errors"
	"net/http"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"
	"github.com/labstack/echo/v5"
)

var errIDPProfileRepositoryUnavailable = errors.New("SAML identity provider profile repository is unavailable")

func requestedIDPProfileID(c *echo.Context) string {
	if profileID := c.Param("profile_id"); profileID != "" {
		return profileID
	}
	return samldomain.DefaultIDPProfileID
}

func (d Deps) resolveIDPProfile(ctx context.Context, tenantID, profileID string) (*samldomain.SamlIdentityProviderProfile, error) {
	if d.IDPProfileRepo == nil {
		return nil, errIDPProfileRepositoryUnavailable
	}
	if profileID == samldomain.DefaultIDPProfileID {
		return d.IDPProfileRepo.EnsureDefaultIDPProfile(ctx, tenantID)
	}
	return d.IDPProfileRepo.FindIDPProfileByID(ctx, tenantID, profileID)
}

func (d Deps) requireIDPProfile(c *echo.Context) (*samldomain.SamlIdentityProviderProfile, error) {
	profile, err := d.resolveIDPProfile(c.Request().Context(), support.RequestTenantID(c), requestedIDPProfileID(c))
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, "SAML identity provider profile not found")
	}
	return profile, nil
}

func idpProfileRoute(profileID, suffix string) string {
	if profileID == samldomain.DefaultIDPProfileID {
		return "/saml/" + suffix
	}
	return "/saml/idp/" + profileID + "/" + suffix
}

func (d Deps) idpProfileURL(c *echo.Context, profileID, suffix string) string {
	return support.TenantURL(c, idpProfileRoute(profileID, suffix), d.Issuer)
}

func (d Deps) idpProfileEntityID(c *echo.Context, profileID string) string {
	if profileID == samldomain.DefaultIDPProfileID {
		return support.RequestIssuer(c, d.Issuer)
	}
	return support.TenantURL(c, "/saml/idp/"+profileID, d.Issuer)
}

func idpProfileSigningContext(c *echo.Context, profileID string) context.Context {
	return samltoken.WithSignerScope(c.Request().Context(), profileID)
}
