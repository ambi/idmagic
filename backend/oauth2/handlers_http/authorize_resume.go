package handlers_http

import (
	"net/http"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// handleFederatedResume resumes the OAuth authorization transaction after an
// external identity provider has established an IdMagic session.
func (d Deps) handleFederatedResume(c *echo.Context) error {
	req, err := d.transactionRequest(c)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	authn, err := d.ResolveAuthentication(c)
	if err != nil || authn == nil || authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "A completed authentication session is required.")
	}
	user, err := d.UserRepo.FindBySub(c.Request().Context(), authn.UserID)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive() {
		return support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "The user is not active.")
	}
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), req.ClientID)
	if err != nil {
		return err
	}
	if client == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_transaction", "The client does not exist.")
	}
	if client.FirstParty {
		if redirected, policyErr := d.enforceDefaultSignInPolicy(c, authn, true); policyErr != nil {
			return policyErr
		} else if redirected {
			return c.Redirect(http.StatusSeeOther, d.pendingAuthPath(c, authn))
		}
	}

	req.UserID, req.AuthTime, req.AMR, req.ACR = &authn.UserID, &authn.AuthTime, authn.AMR, &authn.ACR
	authTime := time.Unix(authn.AuthTime, 0).UTC()
	d.emitAuthenticationSuccess(c, authTime, user, authn, req.ClientID)
	gateNext, err := d.recordLoginAndRequiredAction(c, user, authTime)
	if err != nil {
		return err
	}
	if gateNext != "" {
		return c.Redirect(http.StatusSeeOther, gateNext)
	}
	next, err := d.completeAfterAuthn(c, req, client, authn)
	if err != nil {
		return err
	}
	if next.RedirectTo != "" {
		d.clearTransactionCookie(c)
	}
	return redirectAuthorizationNext(c, next)
}
