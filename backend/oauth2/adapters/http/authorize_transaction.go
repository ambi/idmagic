package http

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func (d Deps) transactionRequest(c *echo.Context) (*domain.AuthorizationRequest, error) {
	cookie, err := c.Cookie(authorizationTransactionCookie)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("認可トランザクションがありません")
	}
	req, err := d.RequestStore.Find(c.Request().Context(), cookie.Value)
	if err != nil {
		return nil, err
	}
	if req == nil || req.TenantID != support.RequestTenantID(c) || time.Now().After(req.ExpiresAt) || req.State != spec.AuthFlowReceived {
		return nil, errors.New("認可トランザクションが無効または期限切れです")
	}
	return req, nil
}

func validReturnTo(c *echo.Context, returnTo string) bool {
	if strings.Contains(returnTo, "\\") {
		return false
	}
	parsed, err := url.Parse(returnTo)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Fragment != "" || path.Clean(parsed.Path) != parsed.Path {
		return false
	}
	adminRoot, wsfedRoot := support.TenantRoute(c, "/admin"), support.TenantRoute(c, "/wsfed")
	return parsed.Path == adminRoot || strings.HasPrefix(parsed.Path, adminRoot+"/") || parsed.Path == wsfedRoot
}

func (d Deps) setTransactionCookie(c *echo.Context, requestID string) {
	c.SetCookie(&http.Cookie{Name: authorizationTransactionCookie, Value: requestID, Path: support.TenantCookiePath(c), Secure: d.SecureCookies(), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 600}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}

func (d Deps) clearTransactionCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{Name: authorizationTransactionCookie, Path: support.TenantCookiePath(c), Secure: d.SecureCookies(), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}

func (d Deps) setSessionCookie(c *echo.Context, sessionID string) {
	c.SetCookie(&http.Cookie{Name: usecases.SessionCookie, Value: sessionID, Path: support.TenantCookiePath(c), Secure: d.SecureCookies(), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: usecases.SessionTTLSeconds}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}

func (d Deps) clearSessionCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{Name: usecases.SessionCookie, Path: support.TenantCookiePath(c), Secure: d.SecureCookies(), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}
