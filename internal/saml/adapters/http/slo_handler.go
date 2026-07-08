package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	samldomain "github.com/ambi/idmagic/internal/saml/domain"
	samlusecases "github.com/ambi/idmagic/internal/saml/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/spec"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
)

// handleSamlSLO は SAML Single Logout を処理する。ローカルセッションを破棄し、要求元 SP の
// SingleLogoutService が登録されていれば許可済みの返送先へリダイレクトする。判定不能・未登録の
// 返送先へはリダイレクトしない (open redirect 防止)。
func (d Deps) handleSamlSLO(c *echo.Context) error {
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	if samlRequest := samlParam(c, "SAMLRequest", ""); samlRequest != "" {
		return d.handleSamlLogoutRequest(c, samlRequest, samlParam(c, "RelayState", ""))
	}
	entityID := samlParam(c, "entityID", "sp")
	relayState := samlParam(c, "RelayState", "")

	if d.SessionManager != nil {
		_ = d.SessionManager.Revoke(ctx, c.Request().Header.Get("Cookie"))
	}
	d.clearSessionCookie(c)
	d.emit(&spec.SamlLogout{At: time.Now().UTC(), TenantID: tenantID, EntityID: entityID})

	if target := d.logoutService().ResolveRedirect(ctx, tenantID, entityID, relayState); target != "" {
		return c.Redirect(http.StatusSeeOther, target)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.String(http.StatusOK, "signed out")
}

func (d Deps) handleSamlLogoutRequest(c *echo.Context, encodedRequest, relayState string) error {
	now := time.Now().UTC()
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)
	binding := samldomain.BindingRedirect
	var (
		xml []byte
		err error
	)
	if c.Request().Method == http.MethodPost {
		binding = samldomain.BindingPOST
		xml, err = samldomain.DecodePost(encodedRequest)
	} else {
		xml, err = samldomain.DecodeRedirect(encodedRequest)
	}
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid LogoutRequest")
	}
	req, err := samldomain.ParseLogoutRequest(xml)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid LogoutRequest")
	}
	if d.SamlSPRepo == nil {
		return c.String(http.StatusBadRequest, "SAML is not available")
	}
	expectedDestination := strings.TrimRight(support.RequestIssuer(c, d.Issuer), "/") + support.TenantRoute(c, "/saml/slo")
	decision, err := d.logoutService().ValidateLogoutRequest(ctx, samlusecases.LogoutRequestInput{
		TenantID:            tenantID,
		Request:             req,
		Binding:             binding,
		RawXML:              xml,
		RawQuery:            c.Request().URL.RawQuery,
		ExpectedDestination: expectedDestination,
	})
	if err != nil {
		return err
	}
	if decision.EmitLogout {
		d.emit(&spec.SamlLogout{At: now, TenantID: tenantID, EntityID: req.Issuer})
	}
	if decision.BadRequest != "" {
		return c.String(http.StatusBadRequest, decision.BadRequest)
	}
	sp := decision.SP

	if d.SessionManager != nil {
		_ = d.SessionManager.Revoke(ctx, c.Request().Header.Get("Cookie"))
	}
	d.clearSessionCookie(c)
	d.emit(&spec.SamlLogout{At: now, TenantID: tenantID, EntityID: req.Issuer})
	response, err := d.buildLogoutResponse(c, *sp, req.ID, now)
	if err != nil {
		return err
	}
	encoded, err := samldomain.EncodeRedirect(response)
	if err != nil {
		return err
	}
	target := sp.SLOURL + "?SAMLResponse=" + url.QueryEscape(encoded)
	if relayState != "" {
		target += "&RelayState=" + url.QueryEscape(relayState)
	}
	return c.Redirect(http.StatusSeeOther, target)
}

func (d Deps) buildLogoutResponse(c *echo.Context, sp spec.SamlServiceProvider, inResponseTo string, now time.Time) ([]byte, error) {
	if d.FederationSigner == nil {
		return nil, fmt.Errorf("SAML signer is required")
	}
	doc := etree.NewDocument()
	resp := doc.CreateElement("samlp:LogoutResponse")
	resp.CreateAttr("xmlns:samlp", "urn:oasis:names:tc:SAML:2.0:protocol")
	resp.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
	resp.CreateAttr("ID", fmt.Sprintf("_logout-%d", now.UnixNano()))
	resp.CreateAttr("Version", "2.0")
	resp.CreateAttr("IssueInstant", now.Format(time.RFC3339))
	resp.CreateAttr("Destination", sp.SLOURL)
	resp.CreateAttr("InResponseTo", inResponseTo)
	resp.CreateElement("saml:Issuer").SetText(support.RequestIssuer(c, d.Issuer))
	status := resp.CreateElement("samlp:Status")
	status.CreateElement("samlp:StatusCode").CreateAttr("Value", "urn:oasis:names:tc:SAML:2.0:status:Success")
	signed, err := d.FederationSigner.Sign(resp, "ID")
	if err != nil {
		return nil, err
	}
	doc.SetRoot(signed)
	doc.Indent(2)
	var b strings.Builder
	if _, err := doc.WriteTo(&b); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

func (d Deps) clearSessionCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure は HTTPS issuer で有効化、ローカル HTTP 開発では意図的に無効。
		Name: authusecases.SessionCookie, Path: support.TenantCookiePath(c),
		Secure: d.SecureCookies(), HttpOnly: true, SameSite: http.SameSiteLaxMode,
		MaxAge: -1,
	})
}

// samlParam はクエリ / フォームから primary を取り、空なら fallback キーを引く。
func samlParam(c *echo.Context, primary, fallback string) string {
	if v := c.QueryParam(primary); v != "" {
		return v
	}
	if v := c.FormValue(primary); v != "" {
		return v
	}
	if fallback != "" {
		if v := c.QueryParam(fallback); v != "" {
			return v
		}
	}
	return ""
}
