package handlers_http

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	mfausecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	approvalusecases "github.com/ambi/idmagic/backend/oauth2/approval/usecases"
	sharedusecases "github.com/ambi/idmagic/backend/oauth2/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

func (d Deps) handleBackchannelAuthenticate(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return writeOAuthError(c, sharedusecases.NewOAuthError("invalid_request", "form parse failed"))
	}
	authed, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	clientIP := extractClientIP(c.Request(), d.TrustedForwardedHops)
	if blocked, rateErr := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "backchannel_authentication", authed.ID+"|"+clientIP); rateErr != nil {
		return rateErr
	} else if blocked {
		return nil
	}
	requestedExpiryValue, requestedExpiryPresent, err := optionalInt(c.Request().PostFormValue("requested_expiry"))
	if err != nil {
		return writeOAuthError(c, sharedusecases.NewOAuthError("invalid_request", "requested_expiry must be an integer"))
	}
	var requestedExpiry *int
	if requestedExpiryPresent {
		requestedExpiry = &requestedExpiryValue
	}
	result, err := approvalusecases.StartApproval(c.Request().Context(), approvalusecases.StartApprovalDeps{
		ClientRepo: d.ClientRepo, UserRepo: d.UserRepo, AgentRepo: d.AgentRepo,
		Store: d.ApprovalRequestStore, HintVerifier: d.IDTokenHintVerifier,
		AuthzDetailTypeRepo: d.AuthzDetailTypeRepo, Notifier: d.Notifier,
		ApprovalURL: strings.TrimSuffix(support.RequestIssuer(c, d.Issuer), "/") + "/account/approvals",
		Emit:        d.Emit,
	}, approvalusecases.StartApprovalInput{
		ClientID: authed.ID, LoginHint: c.Request().PostFormValue("login_hint"),
		IDTokenHint: c.Request().PostFormValue("id_token_hint"), Scope: c.Request().PostFormValue("scope"),
		AuthorizationDetailsRaw: c.Request().PostFormValue("authorization_details"),
		BindingMessage:          c.Request().PostFormValue("binding_message"), RequestedExpiry: requestedExpiry,
	}, time.Now().UTC())
	if err != nil {
		return writeOAuthError(c, err)
	}
	return c.JSON(http.StatusOK, result)
}

type accountApprovalResponse struct {
	ID                   string                     `json:"id"`
	ClientID             string                     `json:"client_id"`
	ClientName           string                     `json:"client_name"`
	AgentName            string                     `json:"agent_name,omitempty"`
	Scopes               []string                   `json:"scopes"`
	AuthorizationDetails []spec.AuthorizationDetail `json:"authorization_details,omitempty"`
	BindingMessage       *string                    `json:"binding_message,omitempty"`
	RequestedAt          time.Time                  `json:"requested_at"`
	ExpiresAt            time.Time                  `json:"expires_at"`
}

func (d Deps) handleListMyApprovalRequests(c *echo.Context) error {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return err
	}
	if authn == nil || authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	records, err := approvalusecases.ListPendingApprovals(c.Request().Context(), d.ApprovalRequestStore, authn.UserID)
	if err != nil {
		return err
	}
	clientIDs := make([]string, len(records))
	for i, rec := range records {
		clientIDs[i] = rec.ClientID
	}
	names := map[string]string{}
	if d.ClientDisplayNameResolver != nil {
		names = d.ClientDisplayNameResolver.ResolveAll(c.Request().Context(), support.RequestTenantID(c), clientIDs)
	}
	response := make([]accountApprovalResponse, 0, len(records))
	for _, rec := range records {
		clientName := names[rec.ClientID]
		if clientName == "" {
			clientName = rec.ClientID
		}
		agentName := ""
		if rec.AgentID != nil && d.AgentRepo != nil {
			if agent, findErr := d.AgentRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), *rec.AgentID); findErr != nil {
				return findErr
			} else if agent != nil {
				agentName = agent.Name
			}
		}
		response = append(response, accountApprovalResponse{
			ID: rec.ID, ClientID: rec.ClientID, ClientName: clientName, AgentName: agentName,
			Scopes: append([]string(nil), rec.Scopes...), AuthorizationDetails: append([]spec.AuthorizationDetail(nil), rec.AuthorizationDetails...),
			BindingMessage: rec.BindingMessage, RequestedAt: rec.RequestedAt, ExpiresAt: rec.ExpiresAt,
		})
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"approval_requests": response})
}

type accountApprovalDecisionRequest struct {
	Decision string `json:"decision"`
}

func (d Deps) handleDecideMyApprovalRequest(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return err
	}
	if authn == nil || authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	if !mfausecases.StepUpSatisfied(authn, time.Now().UTC()) {
		return support.WriteBrowserError(c, http.StatusForbidden, "step_up_required", "This operation requires reauthentication.")
	}
	var input accountApprovalDecisionRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if input.Decision != "approve" && input.Decision != "deny" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "decision must be approve or deny")
	}
	err = approvalusecases.DecideApproval(c.Request().Context(), d.ApprovalRequestStore, d.Emit, authn.UserID, c.Param("id"), input.Decision == "approve", time.Now().UTC())
	if err != nil {
		return writeAccountApprovalError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func writeAccountApprovalError(c *echo.Context, err error) error {
	var oauthErr *sharedusecases.OAuthError
	if !errors.As(err, &oauthErr) {
		return err
	}
	if oauthErr.Code == "access_denied" {
		return support.WriteBrowserError(c, http.StatusForbidden, oauthErr.Code, oauthErr.Description)
	}
	return support.WriteBrowserError(c, http.StatusBadRequest, oauthErr.Code, oauthErr.Description)
}

func optionalInt(raw string) (int, bool, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, false, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false, err
	}
	return value, true, nil
}
