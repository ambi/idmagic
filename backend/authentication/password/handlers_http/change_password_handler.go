package handlers_http

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	mfausecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/password/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
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
	if !mfausecases.StepUpSatisfied(authn, time.Now().UTC()) {
		return support.WriteProblem(c, http.StatusForbidden, "step_up_required", "This operation requires reauthentication.")
	}
	var input changePasswordAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if input.CurrentPassword == "" || input.NewPassword == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The current and new passwords are required.")
	}

	ctx := c.Request().Context()
	snap := resolvePasswordPolicy(ctx, d)
	changed, err := authusecases.ChangePassword(ctx, authusecases.ChangePasswordDeps{
		UserRepo:                d.UserRepo,
		PasswordHasher:          d.PasswordHasher,
		PasswordHistoryRepo:     d.PasswordHistoryRepo,
		BreachedPasswordChecker: d.BreachedPasswordChecker,
		Emit:                    d.LegacyEmit(),
		Policy:                  snap,
	}, authusecases.ChangePasswordInput{
		Sub:             authn.UserID,
		CurrentPassword: input.CurrentPassword,
		NewPassword:     input.NewPassword,
		Now:             time.Now().UTC(),
	})
	switch {
	case err == nil:
		// パスワードが変わった以上、それ以前に成立した第二要素の証明も同時に古くなる
		// ので、記憶済みの端末をすべて失効させる (wi-91)。
		if err := d.RevokeTrustedDevices(
			ctx, changed.TenantID, changed.ID, spec.TrustedDevicePasswordChange,
		); err != nil {
			return err
		}
		c.Response().Header().Set("Cache-Control", "no-store")
		return c.NoContent(http.StatusNoContent)
	case errors.Is(err, authusecases.ErrUserNotFound):
		return support.WriteProblem(c, http.StatusUnauthorized, "authentication_required", "The authenticated session is invalid.")
	case errors.Is(err, authusecases.ErrCurrentPasswordMismatch):
		return support.WriteProblem(c, http.StatusForbidden, "access_denied", "The current password does not match.")
	case errors.Is(err, authusecases.ErrPasswordReused):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "password_reuse", "The new password cannot reuse a recently used password.")
	default:
		var policyErr *authusecases.PasswordPolicyError
		if errors.As(err, &policyErr) {
			violations := make([]string, len(policyErr.Violations))
			for i, v := range policyErr.Violations {
				violations[i] = string(v)
			}
			return support.NoStoreJSON(c, http.StatusBadRequest, map[string]any{
				"error":      "password_policy",
				"message":    "The password does not meet the security requirements.",
				"violations": violations,
			})
		}
		return err
	}
}
