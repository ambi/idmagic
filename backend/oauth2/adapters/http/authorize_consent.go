package http

import (
	"net/http"
	"strings"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
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
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", err.Error())
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || req.UserID == nil || authn.UserID != *req.UserID {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証セッションが一致しません")
	}
	var input consentAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	if input.Action != "allow" {
		_ = d.RequestStore.UpdateState(ctx, req.ID, spec.AuthFlowRejected)
		d.clearTransactionCookie(c)
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: authorizationErrorURL(req, support.RequestIssuer(c, d.Issuer), "access_denied", "")})
	}

	scopes := strings.Fields(req.Scope)
	if d.ConsentRepo != nil {
		now := time.Now().UTC()
		if err := d.ConsentRepo.Save(ctx, support.RequestTenantID(c), &oauthdomain.Consent{
			UserID: authn.UserID, ClientID: req.ClientID,
			Scopes: scopes, State: oauthdomain.ConsentGranted,
			GrantedAt: now, ExpiresAt: now.Add(365 * 24 * time.Hour),
			AuthorizationDetails: req.AuthorizationDetails,
		}); err != nil {
			return err
		}
		if d.Emit != nil {
			d.Emit(&oauthdomain.ConsentGrantedEvent{At: now, TenantID: support.RequestTenantID(c), UserID: authn.UserID, ClientID: req.ClientID, Scopes: scopes})
			if len(req.AuthorizationDetails) > 0 {
				d.Emit(&oauthdomain.AuthorizationDetailsConsented{
					At: now, TenantID: support.RequestTenantID(c), UserID: authn.UserID, ClientID: req.ClientID,
					DetailTypes: oauthdomain.DetailTypes(req.AuthorizationDetails),
				})
			}
		}
	}
	redirectTo, err := d.issueCodeURL(ctx, c, req, authn, time.Unix(authn.AuthTime, 0))
	if err != nil {
		return err
	}
	d.clearTransactionCookie(c)
	return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: redirectTo})
}
