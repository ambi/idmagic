package handlers_http

import (
	"errors"
	"net/http"
	"time"

	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"

	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"

	"github.com/labstack/echo/v5"

	httpdeps "github.com/ambi/idmagic/backend/authentication/internal/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/password/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
)

type changePasswordAPIRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func HandleChangePasswordAPI(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return err
	}
	if authn == nil || authn.AuthenticationPending {
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "An authenticated session is required.")
	}
	// パスワード変更は高 sensitivity 操作。step-up 再認証を要求する。
	if !mfadomain.StepUpSatisfied(authn, time.Now().UTC()) {
		return support.WriteStepUpRequired(c)
	}
	var input changePasswordAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The current and new passwords are required.")
	}

	ctx := c.Request().Context()
	snap := authusecases.ResolveTenantPolicy(ctx)
	_, err = authusecases.ChangePassword(ctx, authusecases.ChangePasswordDeps{
		UserRepo:                d.UserRepo,
		PasswordDelegation:      d.PasswordDelegation,
		PasswordHasher:          d.PasswordHasher,
		PasswordHistoryRepo:     d.PasswordHistoryRepo,
		BreachedPasswordChecker: d.BreachedPasswordChecker,
		TrustedDevices:          d.TrustedDeviceRevoker(),
		Emit:                    d.LegacyEmit(),
		Policy:                  snap,
	}, authusecases.ChangePasswordInput{
		Sub:             authn.UserID,
		CurrentPassword: input.CurrentPassword,
		NewPassword:     input.NewPassword,
		Now:             time.Now().UTC(),
	})
	switch {
	case errors.Is(err, passwordports.ErrPasswordDelegated):
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "Change the password in the source directory.")
	case err == nil:
		c.Response().Header().Set("Cache-Control", "no-store")
		return c.NoContent(http.StatusNoContent)
	case errors.Is(err, authusecases.ErrUserNotFound):
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "The authenticated session is invalid.")
	case errors.Is(err, authusecases.ErrCurrentPasswordMismatch):
		return support.WriteProblem(c, http.StatusForbidden, "access_denied", "The current password does not match.")
	case errors.Is(err, authusecases.ErrPasswordReused):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "password_reuse", "The new password cannot reuse a recently used password.")
	default:
		if policyErr, ok := errors.AsType[*authusecases.PasswordPolicyError](err); ok {
			return writePasswordPolicyProblem(c, policyErr)
		}
		return err
	}
}
