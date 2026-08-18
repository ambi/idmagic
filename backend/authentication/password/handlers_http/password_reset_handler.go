package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/password/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

// resolvePasswordPolicy returns the global defaults merged with the tenant override.
func resolvePasswordPolicy(ctx context.Context, d httpdeps.Deps) authusecases.PasswordPolicySnapshot {
	return authusecases.ResolveTenantPolicy(ctx, d.TenantRepo)
}

type forgotPasswordAPIRequest struct {
	Email string `json:"email"`
}

type resetPasswordAPIRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func HandlePasswordResetContext(d httpdeps.Deps, c *echo.Context) error {
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"csrf_token": csrf})
}

func HandleForgotPasswordAPI(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	var input forgotPasswordAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	clientIP := support.ExtractClientIP(c.Request(), d.TrustedForwardedHops)
	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "password_reset", strings.ToLower(input.Email)+"|"+clientIP); err != nil {
		return err
	} else if blocked {
		return nil
	}
	ttl := time.Duration(authusecases.PasswordResetTokenTTLSeconds) * time.Second
	if err := authusecases.RequestPasswordReset(
		c.Request().Context(),
		authusecases.RequestPasswordResetDeps{
			UserRepo: d.UserRepo, TokenStore: d.PasswordResetTokenStore,
			Notifier: d.Notifier, Emit: d.Emit,
			Issuer: support.RequestIssuer(c, d.Issuer), TokenTTL: ttl,
		},
		authusecases.RequestPasswordResetInput{Email: input.Email, Now: time.Now().UTC()},
	); err != nil {
		return err
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleResetPasswordAPI(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	var input resetPasswordAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if strings.TrimSpace(input.Token) == "" || input.NewPassword == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "A token and a new password are required.")
	}
	snap := resolvePasswordPolicy(c.Request().Context(), d)
	reset, err := authusecases.ResetPasswordWithToken(
		c.Request().Context(),
		authusecases.ResetPasswordWithTokenDeps{
			UserRepo: d.UserRepo, TokenStore: d.PasswordResetTokenStore,
			PasswordHasher: d.PasswordHasher, PasswordHistoryRepo: d.PasswordHistoryRepo,
			BreachedPasswordChecker: d.BreachedPasswordChecker,
			Emit:                    d.Emit, Policy: snap,
		},
		authusecases.ResetPasswordWithTokenInput{
			Token: input.Token, NewPassword: input.NewPassword, Now: time.Now().UTC(),
		},
	)
	switch {
	case err == nil:
		// リセットもパスワード変更と同じく、記憶済みの端末をすべて失効させる (wi-91)。
		if err := d.RevokeTrustedDevices(
			c.Request().Context(), reset.TenantID, reset.ID, spec.TrustedDevicePasswordChange,
		); err != nil {
			return err
		}
		return support.NoStoreJSON(c, http.StatusOK, map[string]string{"status": "ok"})
	case errors.Is(err, authusecases.ErrInvalidResetToken):
		return support.WriteProblem(c, http.StatusGone, "invalid_reset_token", "The reset link is invalid or expired.")
	case errors.Is(err, authusecases.ErrPasswordReused):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "password_reuse", "A recently used password cannot be reused.")
	default:
		var policyErr *authusecases.PasswordPolicyError
		if errors.As(err, &policyErr) {
			violations := make([]string, len(policyErr.Violations))
			for i, violation := range policyErr.Violations {
				violations[i] = string(violation)
			}
			return support.NoStoreJSON(c, http.StatusBadRequest, map[string]any{
				"error": "password_policy", "message": "The password does not meet the security requirements.",
				"violations": violations,
			})
		}
		return err
	}
}
