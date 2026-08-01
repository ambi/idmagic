package handlers_http

import (
	"errors"
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type resetAuthenticatorsRequest struct {
	Targets []spec.AuthenticatorResetTarget `json:"targets"`
}

func authenticatorResetDeps(d httpdeps.Deps) authusecases.AuthenticatorResetDeps {
	return authusecases.AuthenticatorResetDeps{
		MfaEnrollmentDeps: mfaEnrollmentDeps(d),
		RecoveryCodeRepo:  d.RecoveryCodeRepo,
	}
}

func HandleResetUserAuthenticators(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input resetAuthenticatorsRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	result, err := authusecases.ResetUserAuthenticators(
		c.Request().Context(), authenticatorResetDeps(d), actor.ID, c.Param("sub"), input.Targets, time.Now().UTC(),
	)
	if err != nil {
		return writeAuthenticatorResetError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"mfa_enrolled":          result.MfaEnrolled,
		"reenrollment_required": result.ReenrollmentRequired,
		"bypass":                result.Bypass,
	})
}

func writeAuthenticatorResetError(c *echo.Context, err error) error {
	if errors.Is(err, authusecases.ErrAuthenticatorResetNotAllowed) {
		return support.WriteBrowserError(c, http.StatusBadRequest, "authenticator_reset_not_allowed", "The authenticator reset cannot be performed.")
	}
	return err
}
