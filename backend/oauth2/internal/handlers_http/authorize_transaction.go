package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func (d Deps) handleTransaction(c *echo.Context) error {
	req, err := d.transactionRequest(c)
	if err != nil {
		if returnTo := c.QueryParam("return_to"); returnTo != "" {
			if !d.SignIn.ValidReturnTo(c, returnTo) {
				return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "return_to is invalid.")
			}
			csrf, csrfErr := d.EnsureCSRFCookie(c)
			if csrfErr != nil {
				return csrfErr
			}
			authn, _ := d.ResolveAuthentication(c)
			if authn != nil && authn.AuthenticationPending {
				return support.NoStoreJSON(c, http.StatusOK, d.pendingStepTransaction(c, csrf, authn))
			}
			return support.NoStoreJSON(c, http.StatusOK, transactionResponse{Kind: "login", CSRFToken: csrf})
		}
		return support.WriteProblem(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	if req.UserID == nil {
		authn, _ := d.ResolveAuthentication(c)
		if authn != nil && authn.AuthenticationPending {
			return support.NoStoreJSON(c, http.StatusOK, d.pendingStepTransaction(c, csrf, authn))
		}
		return support.NoStoreJSON(c, http.StatusOK, transactionResponse{Kind: "login", CSRFToken: csrf})
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn != nil && authn.AuthenticationPending {
		return support.NoStoreJSON(c, http.StatusOK, d.pendingStepTransaction(c, csrf, authn))
	}
	if authn == nil || authn.UserID != *req.UserID {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "The authentication session does not match.")
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), req.ClientID)
	if err != nil {
		return err
	}
	if client == nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_transaction", "The client does not exist.")
	}
	// 表示名は client_name → Application カタログ名 → client_id の順で解決する (wi-141)。
	// client_id を UUID 化したため、同意画面での UUID 生表示を避ける。
	name := d.ClientDisplayNameResolver.Resolve(
		c.Request().Context(), support.RequestTenantID(c), req.ClientID,
	)
	return support.NoStoreJSON(c, http.StatusOK, transactionResponse{
		Kind: "consent", CSRFToken: csrf, ClientName: name, Scopes: strings.Fields(req.Scope),
		AuthorizationDetails: d.renderConsentDetails(c, req.AuthorizationDetails),
	})
}

func (d Deps) transactionRequest(c *echo.Context) (*domain.AuthorizationRequest, error) {
	cookie, err := c.Cookie(support.TenantCookieName(c, authorizationTransactionCookie))
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

// pendingStepTransaction は、第二要素か MFA の登録を待つセッションの transaction 応答を組み立てる。
func (d Deps) pendingStepTransaction(c *echo.Context, csrf string, authn *authdomain.AuthenticationContext) transactionResponse {
	kind, methods, canRememberDevice := d.SignIn.PendingStep(c, authn)
	return transactionResponse{Kind: kind, CSRFToken: csrf, SecondFactorMethods: methods, CanRememberDevice: canRememberDevice}
}

func (d Deps) setTransactionCookie(c *echo.Context, requestID string) {
	c.SetCookie(&http.Cookie{Name: support.TenantCookieName(c, authorizationTransactionCookie), Value: requestID, Path: support.TenantCookiePath(c), Secure: d.SecureCookies() || support.TenantCookieSecure(c), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 600}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}

func (d Deps) clearTransactionCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{Name: support.TenantCookieName(c, authorizationTransactionCookie), Path: support.TenantCookiePath(c), Secure: d.SecureCookies() || support.TenantCookieSecure(c), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}

func (d Deps) clearSessionCookie(c *echo.Context) {
	c.SetCookie(&http.Cookie{Name: support.TenantCookieName(c, sessiondomain.SessionCookie), Path: support.TenantCookiePath(c), Secure: d.SecureCookies() || support.TenantCookieSecure(c), HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1}) //nolint:gosec // Secure is selected from the configured issuer scheme.
}
