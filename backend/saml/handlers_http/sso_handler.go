package handlers_http

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	samldomain "github.com/ambi/idmagic/backend/saml/domain"
	samlresponse "github.com/ambi/idmagic/backend/saml/responses_saml"
	samlusecases "github.com/ambi/idmagic/backend/saml/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/spec"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
	samltoken "github.com/ambi/idmagic/backend/wsfederation/tokens_saml"

	"github.com/beevik/etree"
	"github.com/labstack/echo/v5"
)

// assertionLifetime は発行 assertion / SAMLResponse の有効期間。
const assertionLifetime = 5 * time.Minute

// handleSamlSSORedirect は HTTP-Redirect binding と IdP-initiated SSO を処理する。
// SAMLRequest があれば SP-initiated、無ければ entityID クエリで IdP-initiated とする。
func (d Deps) handleSamlSSORedirect(c *echo.Context) error {
	samlRequest := c.QueryParam("SAMLRequest")
	relayState := c.QueryParam("RelayState")
	if samlRequest == "" {
		return d.handleIdPInitiated(c, relayState)
	}
	xml, err := samldomain.DecodeRedirect(samlRequest)
	if err != nil {
		return d.rejectSSO(c, "", "decode redirect AuthnRequest", err)
	}
	return d.processAuthnRequest(c, xml, relayState, samldomain.BindingRedirect)
}

// handleSamlSSOPost は HTTP-POST binding の SP-initiated SSO を処理する。
func (d Deps) handleSamlSSOPost(c *echo.Context) error {
	samlRequest := c.FormValue("SAMLRequest")
	relayState := c.FormValue("RelayState")
	if samlRequest == "" {
		return d.rejectSSO(c, "", "missing SAMLRequest", nil)
	}
	xml, err := samldomain.DecodePost(samlRequest)
	if err != nil {
		return d.rejectSSO(c, "", "decode POST AuthnRequest", err)
	}
	return d.processAuthnRequest(c, xml, relayState, samldomain.BindingPOST)
}

// processAuthnRequest は復号済み AuthnRequest を解析し、未認証時のログイン往復をまたいで要求を
// 保つ resume URL を組み立ててから発行判断へ渡す。
func (d Deps) processAuthnRequest(c *echo.Context, xml []byte, relayState string, binding samldomain.Binding) error {
	req, err := samldomain.ParseAuthnRequest(xml)
	if err != nil {
		return d.rejectSSO(c, "", "parse AuthnRequest", err)
	}
	encoded, err := samldomain.EncodeRedirect(xml)
	if err != nil {
		return d.rejectSSO(c, req.Issuer, "encode resume request", err)
	}
	resumeURL := support.TenantRoute(c, "/saml/sso") + "?SAMLRequest=" + url.QueryEscape(encoded)
	if relayState != "" {
		resumeURL += "&RelayState=" + url.QueryEscape(relayState)
	}
	return d.issueForRequest(c, req, relayState, resumeURL, xml, binding)
}

// handleIdPInitiated は entityID 指定の IdP-initiated SSO を処理する。AuthnRequest を伴わないため、
// SP の既定 ACS へ InResponseTo 無しの SAMLResponse を発行する。
func (d Deps) handleIdPInitiated(c *echo.Context, relayState string) error {
	entityID := c.QueryParam("entityID")
	if entityID == "" {
		entityID = c.QueryParam("sp")
	}
	if entityID == "" {
		return d.rejectSSO(c, "", "missing SAMLRequest or entityID", nil)
	}
	req := samldomain.AuthnRequest{Issuer: entityID}
	return d.issueForRequest(c, req, relayState, c.Request().URL.RequestURI(), nil, "")
}

// issueForRequest は wire から usecase 入力を組み立て、発行判断の outcome を HTTP に写す。
func (d Deps) issueForRequest(c *echo.Context, req samldomain.AuthnRequest, relayState, resumeURL string, xml []byte, binding samldomain.Binding) error {
	if d.SamlSPRepo == nil {
		return c.String(http.StatusBadRequest, "SAML is not available")
	}
	ctx := c.Request().Context()
	authn, _ := d.AuthnResolver.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: c.Request().Header})
	expectedDestination := strings.TrimRight(support.RequestIssuer(c, d.Issuer), "/") + support.TenantRoute(c, "/saml/sso")

	outcome, err := d.signInService().Issue(ctx, samlusecases.SignInInput{
		TenantID:            support.RequestTenantID(c),
		Request:             req,
		Binding:             binding,
		RawXML:              xml,
		RawQuery:            c.Request().URL.RawQuery,
		ExpectedDestination: expectedDestination,
		Authn:               authn,
		ClientIP:            d.ClientIP(c.Request()),
	})
	if err != nil {
		return err
	}
	switch outcome.Kind {
	case samlusecases.SignInNeedLogin:
		return c.Redirect(http.StatusSeeOther, loginRedirect(c, resumeURL))
	case samlusecases.SignInRejected:
		return c.String(http.StatusBadRequest, kernel.EnglishErrorText(outcome.Message))
	case samlusecases.SignInForbidden:
		return c.String(http.StatusForbidden, kernel.EnglishErrorText(outcome.Message))
	case samlusecases.SignInProtocolError:
		return d.issueProtocolError(c, outcome, relayState)
	default:
		return d.issueResponse(c, outcome, relayState)
	}
}

func (d Deps) issueProtocolError(c *echo.Context, o samlusecases.SignInOutcome, relayState string) error {
	responseXML, err := samlresponse.BuildErrorResponse(support.RequestIssuer(c, d.Issuer), o.Validated.ACSURL, o.Validated.InResponseTo, o.ProtocolStatus, o.Now)
	if err != nil {
		return d.rejectSSO(c, o.SP.EntityID, "protocol error build failed", err)
	}
	support.SetAutoPostFormCSP(c, o.Validated.ACSURL)
	formHTML, err := samlresponse.EncodePostForm(responseXML, o.Validated.ACSURL, relayState)
	if err != nil {
		return d.rejectSSO(c, o.SP.EntityID, "protocol error form failed", err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.HTML(http.StatusOK, string(formHTML))
}

// issueResponse は発行データから SAMLResponse を組み立て・署名し、自動 POST フォームで返す。
func (d Deps) issueResponse(c *echo.Context, o samlusecases.SignInOutcome, relayState string) error {
	assertion, err := d.buildAssertion(c, o.SP, o.Validated, o.ClaimResult, o.Authn, o.Now)
	if err != nil {
		return d.rejectSSO(c, o.SP.EntityID, "assertion build failed", err)
	}
	responseXML, err := samlresponse.BuildResponse(samlresponse.ResponseInput{
		Issuer:       support.RequestIssuer(c, d.Issuer),
		Destination:  o.Validated.ACSURL,
		InResponseTo: o.Validated.InResponseTo,
		IssueInstant: o.Now,
		Assertion:    assertion,
		SignResponse: o.SP.SignResponse,
	}, d.FederationSigner)
	if err != nil {
		return d.rejectSSO(c, o.SP.EntityID, "response build failed", err)
	}
	// 自動 POST は cross-origin の ACS へ form 送信し固定の submit script を含む。
	// 当該レスポンスの CSP に form-action=ACS と script hash を許可する (ADR-076)。
	support.SetAutoPostFormCSP(c, o.Validated.ACSURL)
	formHTML, err := samlresponse.EncodePostForm(responseXML, o.Validated.ACSURL, relayState)
	if err != nil {
		return d.rejectSSO(c, o.SP.EntityID, "form render failed", err)
	}

	d.emit(&samldomain.SamlSignInIssued{At: o.Now, TenantID: support.RequestTenantID(c), EntityID: o.SP.EntityID, UserID: o.Authn.UserID})
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.HTML(http.StatusOK, string(formHTML))
}

// buildAssertion は claim 発行結果から SAML 2.0 assertion を組み立て、SP 設定に従って署名する。
func (d Deps) buildAssertion(c *echo.Context, sp samldomain.SamlServiceProvider, validated samldomain.ValidatedSignIn, result claimusecases.ClaimIssuanceResult, authn *authdomain.AuthenticationContext, now time.Time) (*etree.Element, error) {
	authnMethod := feddomain.AuthnUnspecified
	if slices.Contains(authn.AMR, "pwd") {
		authnMethod = feddomain.AuthnPassword
	}
	in := samltoken.AssertionInput{
		Version:      samltoken.SAML20,
		Issuer:       support.RequestIssuer(c, d.Issuer),
		Audience:     sp.EffectiveAudience(),
		Recipient:    validated.ACSURL,
		InResponseTo: validated.InResponseTo,
		IssueInstant: now,
		NotBefore:    now.Add(-1 * time.Minute),
		NotOnOrAfter: now.Add(assertionLifetime),
		AuthnInstant: time.Unix(authn.AuthTime, 0).UTC(),
		AuthnMethod:  authnMethod,
		Result:       result,
	}
	if sp.SignAssertion {
		signed, _, err := samltoken.BuildSignedAssertion(in, d.FederationSigner)
		return signed, err
	}
	assertion, _, err := samltoken.BuildAssertion(in)
	return assertion, err
}

func (d Deps) rejectSSO(c *echo.Context, entityID, reason string, cause error) error {
	msg := reason
	if cause != nil {
		msg = reason + ": " + cause.Error()
	}
	d.emit(&samldomain.SamlSignInRejected{At: time.Now().UTC(), TenantID: support.RequestTenantID(c), EntityID: entityID, Reason: msg})
	return c.String(http.StatusBadRequest, kernel.EnglishErrorText(reason))
}

func (d Deps) emit(event spec.DomainEvent) {
	if d.Emit != nil {
		d.Emit(event)
	}
}

// loginRedirect はログイン UI への誘導 URL を、認証後に SAML 要求へ戻す return_to つきで組み立てる
// (同一オリジンの相対パスのみ)。
func loginRedirect(c *echo.Context, returnTo string) string {
	return support.TenantRoute(c, "/login") + "?return_to=" + url.QueryEscape(returnTo)
}
