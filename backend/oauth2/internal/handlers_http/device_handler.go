// /device_authorization + browser device approval API
package handlers_http

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	deviceusecases "github.com/ambi/idmagic/backend/oauth2/internal/device/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type deviceAPIRequest struct {
	UserCode string `json:"user_code"`
	Action   string `json:"action"`
}

func (d Deps) handleDeviceAuthorization(c *echo.Context) error {
	if err := c.Request().ParseForm(); err != nil {
		return c.JSON(http.StatusBadRequest, support.OAuthErrorBody("invalid_request", "form parse"))
	}
	client, err := d.authenticateTokenClient(c)
	if err != nil {
		return writeOAuthError(c, err)
	}
	clientIP := c.RealIP()
	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "device_authorization", client.ID+"|"+clientIP); err != nil {
		return err
	} else if blocked {
		return nil
	}
	in := deviceusecases.DeviceAuthorizationInput{
		ClientID: client.ID,
		Scope:    c.Request().PostFormValue("scope"),
	}
	res, err := deviceusecases.RequestDeviceAuthorization(c.Request().Context(), deviceusecases.DeviceAuthorizationDeps{
		ClientRepo: d.ClientRepo, DeviceCodeStore: d.DeviceCodeStore,
		BaseVerification: support.RequestIssuer(c, d.Issuer) + "/device", Emit: d.Emit,
	}, in, time.Now().UTC())
	if err != nil {
		return writeOAuthError(c, err)
	}
	return c.JSON(http.StatusOK, res)
}

func (d Deps) handleDeviceContext(c *echo.Context) error {
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	userCode := strings.ToUpper(strings.TrimSpace(c.QueryParam("user_code")))
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{
		"kind": "device", "user_code": userCode, "csrf_token": csrf,
	})
}

func (d Deps) handleDeviceAPI(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	var input deviceAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if strings.TrimSpace(input.UserCode) == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "A device code is required.")
	}
	authn, _ := d.AuthnResolver.Resolve(
		c.Request().Context(),
		authdomain.HTTPHeadersAdapter{H: c.Request().Header},
	)
	if authn == nil {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "Authentication is required.")
	}
	if input.Action == "deny" {
		if err := deviceusecases.DenyUserCode(
			c.Request().Context(),
			deviceusecases.VerifyUserCodeDeps{DeviceCodeStore: d.DeviceCodeStore, Emit: d.Emit},
			input.UserCode,
			authn.UserID,
			time.Now().UTC(),
		); err != nil {
			return writeOAuthError(c, err)
		}
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: "/status?state=denied"})
	}
	decision, err := deviceusecases.ApproveUserCode(
		c.Request().Context(),
		deviceusecases.VerifyUserCodeDeps{DeviceCodeStore: d.DeviceCodeStore, Emit: d.Emit},
		d.ApplicationGate,
		deviceusecases.ApproveUserCodeInput{UserCode: input.UserCode, Authn: authn, ClientIP: c.RealIP()},
		time.Now().UTC(),
	)
	if err != nil {
		return writeOAuthError(c, err)
	}
	if decision.StepUpRequired {
		if len(d.SignIn.SecondFactorMethods(c, authn.UserID)) > 0 {
			pending, err := d.SignIn.RequireSecondFactor(c.Request().Context(), c, authn.SessionID)
			if err != nil {
				return err
			}
			if pending == nil {
				return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "The session has expired.")
			}
			returnTo := support.TenantRoute(c, "/device") + "?user_code=" + url.QueryEscape(input.UserCode)
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{
				Next: d.SignIn.PendingPath(c, pending) + "?return_to=" + url.QueryEscape(returnTo),
			})
		}
		return support.WriteProblem(c, http.StatusForbidden, "access_denied", "The application's sign-in policy requirements were not met.")
	}
	if !decision.Allowed {
		return support.WriteProblem(c, http.StatusForbidden, "access_denied", "The user is not allowed to access this application.")
	}
	return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: "/status?state=approved"})
}
