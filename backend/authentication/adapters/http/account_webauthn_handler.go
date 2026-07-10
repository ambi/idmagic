// /api/account/mfa/webauthn/* — エンドユーザー自身の WebAuthn / Passkey の self-service
// 登録・解除 (wi-26 / ADR-087)。登録は attestation 検証 (RP ID / origin / challenge) を所持
// 証明とし、解除は step-up 再認証 (ADR-043) を要求する。
package http

import (
	"encoding/json"
	"net/http"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type webAuthnRegistrationFinishRequest struct {
	Attestation json.RawMessage `json:"attestation"`
	Label       *string         `json:"label,omitempty"`
}

type webAuthnCredentialRemoveRequest struct {
	CredentialID string `json:"credential_id"`
}

func (d Deps) webAuthnAccountDeps() authusecases.WebAuthnDeps {
	return authusecases.WebAuthnDeps{
		RP:             d.WebAuthnRP,
		UserRepo:       d.UserRepo,
		CredentialRepo: d.WebAuthnCredentialRepo,
		MfaFactorRepo:  d.MfaFactorRepo,
		SessionStore:   d.WebAuthnSessionStore,
		Emit:           d.Emit,
	}
}

func (d Deps) handleStartWebAuthnRegistration(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAuthenticatedSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	creation, err := authusecases.StartWebAuthnRegistration(c.Request().Context(), d.webAuthnAccountDeps(), sub)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, creation)
}

func (d Deps) handleFinishWebAuthnRegistration(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireAuthenticatedSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	var input webAuthnRegistrationFinishRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if err := authusecases.FinishWebAuthnRegistration(
		c.Request().Context(), d.webAuthnAccountDeps(), sub, []byte(input.Attestation), input.Label, time.Now().UTC(),
	); err != nil {
		return d.writeAccountError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func (d Deps) handleRemoveWebAuthnCredential(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// Passkey の解除は高 sensitivity 操作。step-up 再認証を要求する (ADR-043)。
	sub, err := d.requireStepUpSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	if d.WebAuthnCredentialRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "webauthn_unavailable", "パスキー認証は利用できません")
	}
	var input webAuthnCredentialRemoveRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if err := authusecases.RemoveWebAuthnCredential(
		c.Request().Context(), d.webAuthnAccountDeps(), sub, input.CredentialID, time.Now().UTC(),
	); err != nil {
		return d.writeAccountError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
