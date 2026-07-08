package http

import (
	"net/http"
	"net/url"
	"time"

	authdomain "github.com/ambi/idmagic/internal/authentication/domain"
	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"
	"github.com/ambi/idmagic/internal/shared/spec"
	"github.com/ambi/idmagic/internal/wsfederation/adapters/samltoken"
	"github.com/ambi/idmagic/internal/wsfederation/adapters/wsfed"
	feddomain "github.com/ambi/idmagic/internal/wsfederation/domain"
	wsfedusecases "github.com/ambi/idmagic/internal/wsfederation/usecases"

	"github.com/labstack/echo/v5"
)

// assertionLifetime は発行 assertion / RSTR の有効期間。
const assertionLifetime = 5 * time.Minute

// samlVersion は RP の token type を samltoken の SAML バージョンへ写す。
func samlVersion(t spec.WsFedTokenType) samltoken.SAMLVersion {
	if t == spec.TokenTypeSAML20 {
		return samltoken.SAML20
	}
	return samltoken.SAML11
}

// handleWsFed は WS-Federation passive エンドポイントを wa で分岐する。
func (d Deps) handleWsFed(c *echo.Context) error {
	req := feddomain.ParseSignInRequest(c.QueryParam)
	switch req.Wa {
	case feddomain.WaSignIn:
		return d.handleWsFedSignIn(c, req)
	case feddomain.WaSignOut, feddomain.WaSignOutCleanup:
		return d.handleWsFedSignOut(c, req)
	default:
		return c.String(http.StatusBadRequest, "unsupported wa")
	}
}

// handleWsFedSignIn は wire から usecase 入力を組み立て、発行判断の outcome を HTTP に写す。
func (d Deps) handleWsFedSignIn(c *echo.Context, req feddomain.WsFedSignInRequest) error {
	ctx := c.Request().Context()
	authn, _ := d.AuthnResolver.Resolve(ctx, authdomain.HTTPHeadersAdapter{H: c.Request().Header})

	outcome, err := d.signInService().Issue(ctx, wsfedusecases.SignInInput{
		TenantID: support.RequestTenantID(c),
		Request:  req,
		Authn:    authn,
		ClientIP: d.ClientIP(c.Request()),
	})
	if err != nil {
		return err
	}
	switch outcome.Kind {
	case wsfedusecases.SignInNeedLogin:
		return c.Redirect(http.StatusSeeOther, loginRedirect(c))
	case wsfedusecases.SignInRejected:
		return c.String(outcome.Status, outcome.Message)
	case wsfedusecases.SignInForbidden:
		return c.String(http.StatusForbidden, outcome.Message)
	default:
		return d.issuePassiveForm(c, outcome)
	}
}

// issuePassiveForm は発行データから署名済み SAML assertion を RSTR に包み、自動 POST で返す。
func (d Deps) issuePassiveForm(c *echo.Context, o wsfedusecases.SignInOutcome) error {
	tenantID := support.RequestTenantID(c)
	rp := o.Validated.RelyingParty

	signed, _, err := samltoken.BuildSignedAssertion(samltoken.AssertionInput{
		Version:      samlVersion(o.TokenType),
		Issuer:       support.RequestIssuer(c, d.Issuer),
		Audience:     rp.EffectiveAudience(),
		Recipient:    o.Validated.ReplyURL,
		IssueInstant: o.Now,
		NotBefore:    o.Now.Add(-1 * time.Minute),
		NotOnOrAfter: o.Now.Add(assertionLifetime),
		AuthnInstant: time.Unix(o.Authn.AuthTime, 0).UTC(),
		AuthnMethod:  o.AuthnMethod,
		Result:       o.ClaimResult,
	}, d.FederationSigner)
	if err != nil {
		return c.String(http.StatusInternalServerError, "assertion build failed")
	}

	rstr, err := wsfed.BuildRSTR(signed, rp.Wtrealm, string(o.TokenType), o.Now, o.Now.Add(assertionLifetime))
	if err != nil {
		return c.String(http.StatusInternalServerError, "rstr build failed")
	}
	wresult, err := wsfed.SerializeRSTR(rstr)
	if err != nil {
		return c.String(http.StatusInternalServerError, "rstr serialize failed")
	}
	// 自動 POST は cross-origin の ReplyURL へ form 送信し固定の submit script を含む。
	// 当該レスポンスの CSP に form-action=ReplyURL と script hash を許可する (ADR-076)。
	support.SetAutoPostFormCSP(c, o.Validated.ReplyURL)
	formHTML, err := wsfed.RenderPassiveForm(o.Validated.ReplyURL, wresult, o.Validated.Wctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, "form render failed")
	}

	d.emit(&spec.WsFedSignInIssued{At: o.Now, TenantID: tenantID, Wtrealm: rp.Wtrealm, UserID: o.Authn.UserID})
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.HTML(http.StatusOK, string(formHTML))
}

// handleWsFedSignOut はローカルセッションを破棄する。wsignout1.0 は許可済み wreply への
// リダイレクトまで行い、wsignoutcleanup1.0 は破棄のみで 200 を返す。
func (d Deps) handleWsFedSignOut(c *echo.Context, req feddomain.WsFedSignInRequest) error {
	ctx := c.Request().Context()
	tenantID := support.RequestTenantID(c)

	if d.SessionManager != nil {
		_ = d.SessionManager.Revoke(ctx, c.Request().Header.Get("Cookie"))
	}
	d.clearSessionCookie(c)
	d.emit(&spec.WsFedSignOut{At: time.Now().UTC(), TenantID: tenantID, Wtrealm: req.Wtrealm})

	if req.Wa == feddomain.WaSignOut {
		if target := d.signOutService().ResolveReply(ctx, tenantID, req); target != "" {
			return c.Redirect(http.StatusSeeOther, target)
		}
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.String(http.StatusOK, "signed out")
}

func (d Deps) emit(event spec.DomainEvent) {
	if d.Emit != nil {
		d.Emit(event)
	}
}

func (d Deps) clearSessionCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{ //nolint:gosec // Secure は HTTPS issuer で有効化、ローカル HTTP 開発では意図的に無効。
		Name: authusecases.SessionCookie, Path: support.TenantCookiePath(c),
		Secure: d.SecureCookies(), HttpOnly: true, SameSite: http.SameSiteLaxMode,
		MaxAge: -1,
	})
}

// loginRedirect はログイン UI への誘導 URL を、認証後に現在の WS-Fed 要求へ戻す
// return_to つきで組み立てる (同一オリジンの相対パスのみ)。
func loginRedirect(c *echo.Context) string {
	returnTo := c.Request().URL.RequestURI()
	return support.TenantRoute(c, "/login") + "?return_to=" + url.QueryEscape(returnTo)
}
