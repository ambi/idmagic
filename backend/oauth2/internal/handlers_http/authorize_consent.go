package handlers_http

import (
	"net/http"
	"strings"
	"time"

	consentusecases "github.com/ambi/idmagic/backend/oauth2/internal/consent/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type consentAPIRequest struct {
	Action string `json:"action"`
}

func (d Deps) handleConsentAPI(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	req, err := d.transactionRequest(c)
	if err != nil {
		return support.WriteProblem(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || req.UserID == nil || authn.UserID != *req.UserID {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "The authentication session does not match.")
	}
	var input consentAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	if input.Action != "allow" {
		_ = d.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowRejected)
		d.clearTransactionCookie(c)
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: authorizationErrorURL(req, support.RequestIssuer(c, d.Issuer), "access_denied", "")})
	}

	if d.ConsentRepo != nil {
		if _, err := consentusecases.GrantConsent(ctx, d.ConsentDeps(), support.RequestTenantID(c), consentusecases.ConsentGrant{
			UserID: authn.UserID, ClientID: req.ClientID,
			Scopes: strings.Fields(req.Scope), AuthorizationDetails: req.AuthorizationDetails,
		}, time.Now().UTC()); err != nil {
			return err
		}
	}
	redirectTo, err := d.issueCodeURL(ctx, c, req, authn, time.Unix(authn.AuthTime, 0))
	if err != nil {
		return err
	}
	d.clearTransactionCookie(c)
	return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: redirectTo})
}
