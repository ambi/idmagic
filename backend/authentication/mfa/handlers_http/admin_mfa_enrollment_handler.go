package handlers_http

import (
	"errors"
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type issueMfaEnrollmentBypassRequest struct {
	ExpiresInSeconds int `json:"expires_in_seconds"`
}

func mfaEnrollmentDeps(d httpdeps.Deps) authusecases.MfaEnrollmentDeps {
	return authusecases.MfaEnrollmentDeps{
		UserRepo: d.UserRepo, MfaFactorRepo: d.MfaFactorRepo,
		WebAuthnCredentialRepo: d.WebAuthnCredentialRepo,
		BypassRepo:             d.MfaEnrollmentBypassRepo, Emit: d.Emit,
	}
}

func HandleIssueMfaEnrollmentBypass(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input issueMfaEnrollmentBypassRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if input.ExpiresInSeconds == 0 {
		input.ExpiresInSeconds = 900
	}
	bypass, err := authusecases.IssueMfaEnrollmentBypass(
		c.Request().Context(), mfaEnrollmentDeps(d), actor.ID, c.Param("sub"),
		time.Duration(input.ExpiresInSeconds)*time.Second, time.Now().UTC(),
	)
	if err != nil {
		return writeMfaEnrollmentAdminError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, map[string]any{"bypass": bypass})
}

func HandleRevokeMfaEnrollmentBypass(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := authusecases.RevokeMfaEnrollmentBypass(c.Request().Context(), mfaEnrollmentDeps(d), actor.ID, c.Param("sub"), time.Now().UTC()); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func writeMfaEnrollmentAdminError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, authusecases.ErrMfaEnrollmentAlreadyComplete):
		return support.WriteBrowserError(c, http.StatusConflict, "mfa_already_enrolled", "The target user is already enrolled in MFA.")
	case errors.Is(err, authusecases.ErrMfaEnrollmentNotAllowed):
		return support.WriteBrowserError(c, http.StatusBadRequest, "mfa_enrollment_not_allowed", "An MFA enrollment bypass cannot be issued.")
	default:
		return err
	}
}
