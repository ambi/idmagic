// /api/account/email/* — primary email の変更と新アドレスの再検証 (self-service, wi-21)。
package handlers_http

import (
	"errors"
	"net/http"
	"strings"
	"time"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type emailChangeRequest struct {
	NewEmail string `json:"new_email"`
}

type emailChangeVerifyRequest struct {
	Token string `json:"token"`
}

// handleEmailVerifyContext は未認証で開かれうる検証ページ用に CSRF トークンを発行する
// (handlePasswordResetContext と同方針)。
func HandleEmailVerifyContext(d Deps, c *echo.Context) error {
	csrf, err := d.EnsureCSRFCookie(c)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"csrf_token": csrf})
}

func HandleRequestEmailChange(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// primary email の変更は高 sensitivity 操作。step-up 再認証を要求する (ADR-043)。
	sub, err := requireStepUpSub(d, c)
	if err != nil {
		return writeAccountError(c, err)
	}
	var input emailChangeRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	err = userusecases.RequestEmailChange(c.Request().Context(), userusecases.RequestEmailChangeDeps{
		UserRepo: d.UserRepo, TokenStore: d.EmailChangeTokenStore,
		EmailSender: d.EmailSender, Emit: d.Emit, Issuer: support.RequestIssuer(c, d.Issuer),
	}, userusecases.RequestEmailChangeInput{Sub: sub, NewEmail: input.NewEmail, Now: time.Now().UTC()})
	if err != nil {
		return writeEmailChangeError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleConfirmEmailChange(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	var input emailChangeVerifyRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	if strings.TrimSpace(input.Token) == "" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "A token is required.")
	}
	if _, err := userusecases.ConfirmEmailChange(c.Request().Context(), userusecases.ConfirmEmailChangeDeps{
		UserRepo: d.UserRepo, TokenStore: d.EmailChangeTokenStore, Emit: d.Emit,
	}, userusecases.ConfirmEmailChangeInput{Token: input.Token, Now: time.Now().UTC()}); err != nil {
		return writeEmailChangeError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]string{"status": "ok"})
}

func writeEmailChangeError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, userusecases.ErrInvalidEmail):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_email", "The email address format is invalid.")
	case errors.Is(err, userusecases.ErrEmailUnchanged):
		return support.WriteBrowserError(c, http.StatusBadRequest, "email_unchanged", "The email address is unchanged.")
	case errors.Is(err, userusecases.ErrEmailTaken):
		return support.WriteBrowserError(c, http.StatusConflict, "email_taken", "This email address is already in use.")
	case errors.Is(err, userusecases.ErrInvalidEmailChangeToken):
		return support.WriteBrowserError(c, http.StatusGone, "invalid_email_change_token", "The confirmation link is invalid or expired.")
	default:
		return writeAccountError(c, err)
	}
}
