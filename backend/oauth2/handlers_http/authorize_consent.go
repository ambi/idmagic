package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"

	"github.com/labstack/echo/v5"
)

type consentAPIRequest struct {
	Action string `json:"action"`
}

// shouldConsumeConsentQuota reports whether granting a consent for (sub,
// clientID) should consume a tenant's consents Hard Quota slot. ConsentRepo.
// Save is an upsert keyed by (tenant, sub, client): only a brand-new or
// previously-Revoked consent should consume a slot, so re-consenting to an
// already-Granted record (e.g. a scope change) doesn't double-count
// (wi-160, ADR-134).
func shouldConsumeConsentQuota(existing *oauthdomain.Consent) bool {
	return existing == nil || existing.State == oauthdomain.ConsentRevoked
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
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "The authentication session does not match.")
	}
	var input consentAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
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
		tenantID := support.RequestTenantID(c)
		if d.QuotaRepo != nil {
			existing, err := d.ConsentRepo.Find(ctx, tenantID, authn.UserID, req.ClientID)
			if err != nil {
				return err
			}
			if shouldConsumeConsentQuota(existing) {
				if err := tenancyusecases.CheckQuotaAndIncrement(ctx, d.QuotaRepo, tenantID, tenancydomain.ResourceConsents, 1); err != nil {
					if qErr, ok := errors.AsType[*tenancydomain.QuotaExceededError](err); ok && d.Emit != nil {
						d.Emit(&tenancydomain.QuotaExceeded{At: now, TenantID: tenantID, Resource: qErr.Resource, HardLimit: true})
					}
					return err
				}
			}
		}
		if err := d.ConsentRepo.Save(ctx, tenantID, &oauthdomain.Consent{
			UserID: authn.UserID, ClientID: req.ClientID,
			Scopes: scopes, State: oauthdomain.ConsentGranted,
			GrantedAt: now, ExpiresAt: now.Add(365 * 24 * time.Hour),
			AuthorizationDetails: req.AuthorizationDetails,
		}); err != nil {
			return err
		}
		if d.Emit != nil {
			d.Emit(&oauthdomain.ConsentGrantedEvent{At: now, TenantID: tenantID, UserID: authn.UserID, ClientID: req.ClientID, Scopes: scopes})
			if len(req.AuthorizationDetails) > 0 {
				d.Emit(&oauthdomain.AuthorizationDetailsConsented{
					At: now, TenantID: tenantID, UserID: authn.UserID, ClientID: req.ClientID,
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
