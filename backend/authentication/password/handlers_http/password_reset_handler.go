package handlers_http

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	"github.com/ambi/idmagic/backend/authentication/password/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/password/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

// resolvePasswordPolicy は global default + tenant override を合成した snapshot を返す。
// テナント解決失敗時はサイレントに global default にフォールバックする
// (パスワードポリシーで認証経路を落とすのは過剰)。
func resolvePasswordPolicy(ctx context.Context, d httpdeps.Deps) authusecases.PasswordPolicySnapshot {
	defaults := domain.PasswordPolicySnapshot{
		MinLength:    authusecases.PasswordPolicyMinLength,
		MaxLength:    authusecases.PasswordPolicyMaxLength,
		HistoryDepth: authusecases.PasswordPolicyHistoryDepth,
	}
	var tenant *tenancydomain.Tenant
	if d.TenantRepo != nil {
		if id := tenancy.TenantID(ctx); id != "" {
			if found, err := d.TenantRepo.FindByID(ctx, id); err == nil {
				tenant = found
			}
		}
	}
	resolved := domain.ResolvePasswordPolicy(tenant, defaults)
	return authusecases.PasswordPolicySnapshot{
		MinLength:    resolved.MinLength,
		MaxLength:    resolved.MaxLength,
		HistoryDepth: resolved.HistoryDepth,
	}
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ttl := time.Duration(authusecases.PasswordResetTokenTTLSeconds) * time.Second
	if err := authusecases.RequestPasswordReset(
		c.Request().Context(),
		authusecases.RequestPasswordResetDeps{
			UserRepo: d.UserRepo, TokenStore: d.PasswordResetTokenStore,
			EmailSender: d.EmailSender, Emit: d.Emit,
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
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if strings.TrimSpace(input.Token) == "" || input.NewPassword == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "A token and a new password are required.")
	}
	snap := resolvePasswordPolicy(c.Request().Context(), d)
	_, err := authusecases.ResetPasswordWithToken(
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
		return support.NoStoreJSON(c, http.StatusOK, map[string]string{"status": "ok"})
	case errors.Is(err, authusecases.ErrInvalidResetToken):
		return support.WriteBrowserError(c, http.StatusGone, "invalid_reset_token", "The reset link is invalid or expired.")
	case errors.Is(err, authusecases.ErrPasswordReused):
		return support.WriteBrowserError(c, http.StatusBadRequest, "password_reuse", "A recently used password cannot be reused.")
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
