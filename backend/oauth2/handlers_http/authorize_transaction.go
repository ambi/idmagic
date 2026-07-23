package handlers_http

import (
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	usecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func (d Deps) handleTransaction(c *echo.Context) error {
	req, err := d.transactionRequest(c)
	if err != nil {
		if returnTo := c.QueryParam("return_to"); returnTo != "" {
			if !validReturnTo(c, returnTo) {
				return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to is invalid.")
			}
			csrf, csrfErr := d.EnsureCSRFCookie(c)
			if csrfErr != nil {
				return csrfErr
			}
			authn, _ := d.ResolveAuthentication(c)
			if authn != nil && authn.AuthenticationPending {
				return support.NoStoreJSON(c, http.StatusOK, d.secondFactorTransaction(c, csrf, authn))
			}
			return support.NoStoreJSON(c, http.StatusOK, transactionResponse{Kind: "login", CSRFToken: csrf})
		}
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	if req.UserID == nil {
		authn, _ := d.ResolveAuthentication(c)
		if authn != nil && authn.AuthenticationPending {
			return support.NoStoreJSON(c, http.StatusOK, d.secondFactorTransaction(c, csrf, authn))
		}
		return support.NoStoreJSON(c, http.StatusOK, transactionResponse{Kind: "login", CSRFToken: csrf})
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn != nil && authn.AuthenticationPending {
		return support.NoStoreJSON(c, http.StatusOK, d.secondFactorTransaction(c, csrf, authn))
	}
	if authn == nil || authn.UserID != *req.UserID {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "The authentication session does not match.")
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), req.ClientID)
	if err != nil {
		return err
	}
	if client == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_transaction", "The client does not exist.")
	}
	// 表示名は client_name → Application カタログ名 → client_id の順で解決する (wi-141)。
	// ADR-084 で client_id を UUID 化したため、同意画面での UUID 生表示を避ける。
	name := d.ClientDisplayNameResolver.Resolve(
		c.Request().Context(), support.RequestTenantID(c), req.ClientID,
	)
	return support.NoStoreJSON(c, http.StatusOK, transactionResponse{
		Kind: "consent", CSRFToken: csrf, ClientName: name, Scopes: strings.Fields(req.Scope),
		AuthorizationDetails: d.renderConsentDetails(c, req.AuthorizationDetails),
	})
}

func (d Deps) transactionRequest(c *echo.Context) (*domain.AuthorizationRequest, error) {
	cookie, err := c.Cookie(authorizationTransactionCookie)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("no authorization transaction is available")
	}
	req, err := d.RequestStore.Find(c.Request().Context(), cookie.Value)
	if err != nil {
		return nil, err
	}
	if req == nil || req.TenantID != support.RequestTenantID(c) || time.Now().After(req.ExpiresAt) || req.State != spec.AuthFlowReceived {
		return nil, errors.New("the authorization transaction is invalid or has expired")
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
