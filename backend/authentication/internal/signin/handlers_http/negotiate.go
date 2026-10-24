package handlers_http

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	negotiatedomain "github.com/ambi/idmagic/backend/authentication/internal/negotiate/domain"
	negotiateusecases "github.com/ambi/idmagic/backend/authentication/internal/negotiate/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/labstack/echo/v5"
)

const negotiateAttemptCookie = "idmagic_negotiate"

type negotiateResult struct {
	RedirectTo string `json:"redirect_to,omitempty"`
}

func (d Deps) negotiateEnabled(c *echo.Context) bool {
	issuer, err := url.Parse(support.RequestIssuer(c, d.Issuer))
	return err == nil && issuer.Scheme == "https" && d.NegotiateService.AcceptsHostname(support.RequestTenantID(c), issuer.Hostname())
}

func (d Deps) handleNegotiateAvailability(c *echo.Context) error {
	enabled := d.NegotiateService != nil && d.negotiateEnabled(c)
	automatic := enabled && d.AutomaticNegotiate && d.NegotiateService.Automatic(support.RequestTenantID(c), d.ClientIP(c.Request()))
	return support.NoStoreJSON(c, http.StatusOK, struct {
		Enabled   bool `json:"enabled"`
		Automatic bool `json:"automatic"`
	}{enabled, automatic})
}

func (d Deps) handleNegotiate(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	var input struct {
		ReturnTo string `json:"return_to"`
	}
	if err := support.DecodeJSON(c.Request(), &input); err != nil || input.ReturnTo != "" && !validReturnTo(c, input.ReturnTo) {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "Invalid login continuation.")
	}
	if input.ReturnTo == "" {
		if _, err := d.Authorizations.Find(c); err != nil {
			return c.String(http.StatusUnauthorized, "Authorization transaction unavailable")
		}
	}
	redirect, handled, err := d.negotiate(c, input.ReturnTo)
	if handled || err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, negotiateResult{RedirectTo: redirect})
}

// NegotiatePassive は WS-Fed の検証済みの要求から呼ぶ。required は RP が WIA を明示したことを表す。
func (d Deps) NegotiatePassive(c *echo.Context, returnTo string, required bool) (bool, error) {
	if d.NegotiateService == nil || !d.negotiateEnabled(c) || !required &&
		(!d.AutomaticNegotiate || !d.NegotiateService.Automatic(support.RequestTenantID(c), d.ClientIP(c.Request()))) {
		return false, nil
	}
	redirect, handled, err := d.negotiate(c, returnTo)
	if handled || err != nil {
		return true, err
	}
	if redirect == "" {
		return false, nil
	}
	return true, c.Redirect(http.StatusSeeOther, redirect)
}

// negotiate はチャレンジとセッションの作成を共通化し、JSON とパッシブの継続先だけを呼び出し側へ返す。
func (d Deps) negotiate(c *echo.Context, returnTo string) (string, bool, error) {
	if d.NegotiateService == nil || !d.negotiateEnabled(c) {
		return "", false, nil
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "login", d.ClientIP(c.Request())); blocked || err != nil {
		return "", true, err
	}
	tenantID, now := support.RequestTenantID(c), time.Now().UTC()
	// CSRF と認可のトランザクションを含め、別のログイン試行へ Cookie を流用しない。
	var key strings.Builder
	key.WriteString(c.Request().URL.Path + "\x00" + returnTo)
	for _, name := range []string{support.CSRFCookie, "idmagic_transaction"} {
		if cookie, err := c.Cookie(support.TenantCookieName(c, name)); err == nil {
			key.WriteString("\x00" + cookie.Value)
		}
	}
	digest := sha256.Sum256([]byte(key.String()))
	target := hex.EncodeToString(digest[:])
	cookie, cookieErr := c.Cookie(support.TenantCookieName(c, negotiateAttemptCookie))
	authorization := c.Request().Header.Get("Authorization")
	validAttempt := cookieErr == nil && negotiatedomain.ValidAttempt(cookie.Value, d.NegotiateAttemptSecret, tenantID, target, now)
	if !validAttempt {
		if cookieErr == nil && authorization != "" {
			return "", true, c.String(http.StatusUnauthorized, "Invalid Negotiate attempt")
		}
		value := (negotiatedomain.Attempt{TenantID: tenantID, Target: target, Expires: now.Add(5 * time.Minute).Unix()}).Sign(d.NegotiateAttemptSecret)
		c.SetCookie(&http.Cookie{
			Name: support.TenantCookieName(c, negotiateAttemptCookie), Value: value,
			Path: support.TenantCookiePath(c), Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 300,
		})
		c.Response().Header().Set("WWW-Authenticate", "Negotiate")
		return "", true, c.String(http.StatusUnauthorized, "Negotiate")
	}
	parts := strings.Fields(authorization)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Negotiate") {
		return "", false, nil
	}
	if len(parts[1]) > base64.StdEncoding.EncodedLen(64*1024) {
		d.emitNegotiate(&negotiateusecases.Rejected{At: now, TenantID: tenantID, Reason: "encoding"})
		return "", true, c.String(http.StatusUnauthorized, "Negotiate authentication rejected")
	}
	token, err := base64.StdEncoding.Strict().DecodeString(parts[1])
	if err != nil || len(token) == 0 || len(token) > 64*1024 {
		d.emitNegotiate(&negotiateusecases.Rejected{At: now, TenantID: tenantID, Reason: "encoding"})
		return "", true, c.String(http.StatusUnauthorized, "Negotiate authentication rejected")
	}
	user, reply, err := d.NegotiateService.Authenticate(c.Request().Context(), tenantID, token, d.ClientIP(c.Request()), now, d.emitNegotiate)
	if errors.Is(err, negotiateusecases.ErrRejected) {
		return "", true, c.String(http.StatusUnauthorized, "Negotiate authentication rejected")
	}
	if err != nil {
		return "", true, err
	}
	authn, err := d.SessionManager.Create(c.Request().Context(), user.ID, []string{authdomain.AMRKerberos}, now)
	if err != nil {
		return "", true, err
	}
	if len(reply) > 0 {
		c.Response().Header().Set("WWW-Authenticate", "Negotiate "+base64.StdEncoding.EncodeToString(reply))
	}
	d.emitNegotiate(&negotiateusecases.Succeeded{At: now, TenantID: tenantID, UserID: user.ID})
	pending, err := d.enforceDefaultSignInPolicy(c, authn, true)
	if err != nil {
		_ = d.SessionManager.Revoke(c.Request().Context(), support.TenantCookieName(c, "idmagic_session")+"="+authn.SessionID)
		return "", true, err
	}
	d.setSessionCookie(c, authn.SessionID)
	if pending {
		path := d.pendingAuthPath(c, authn)
		if returnTo != "" {
			path += "?return_to=" + url.QueryEscape(returnTo)
		}
		return path, false, nil
	}
	gate, err := (Steps{Deps: d}).CompleteSignIn(c, user, authn, "", now)
	if err != nil {
		return "", true, err
	}
	if gate != "" {
		return gate, false, nil
	}
	if returnTo != "" {
		return returnTo, false, nil
	}
	return support.TenantRoute(c, "/authorize/resume"), false, nil
}

func (d Deps) emitNegotiate(event spec.DomainEvent) {
	if d.Emit != nil {
		d.Emit(event)
	}
}
