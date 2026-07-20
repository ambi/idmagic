// /api/account/step_up — 高 sensitivity な self-service 操作のための step-up 再認証
// (ADR-043 / wi-43)。start は利用可能な factor を返し、complete は password / TOTP を
// 検証して session に step_up_at を刻む。sensitive ハンドラは httpdeps.RequireStepUpSub /
// httpdeps.RequireStepUpSession を前段ゲートに使い、recency 窓を外れていれば
// 403 step_up_required。
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/authentication/adapters/http/httpdeps"
	authusecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	webauthnhttp "github.com/ambi/idmagic/backend/authentication/webauthn/adapters/http"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type StepUpStartResponse struct {
	Methods []string `json:"methods"`
}

type stepUpCompleteRequest struct {
	Method    string          `json:"method"`
	Password  string          `json:"password"`
	Code      string          `json:"code"`
	Assertion json.RawMessage `json:"assertion,omitempty"`
}

func stepUpDeps(d httpdeps.Deps) authusecases.StepUpDeps {
	return authusecases.StepUpDeps{
		UserRepo:         d.UserRepo,
		PasswordHasher:   d.PasswordHasher,
		MfaFactorRepo:    d.MfaFactorRepo,
		WebAuthn:         webauthnhttp.WebAuthnAccountDeps(d),
		RecoveryCodeRepo: d.RecoveryCodeRepo,
		SessionManager:   d.SessionManager,
		Emit:             d.Emit,
	}
}

// HandleStepUpWebAuthnChallenge は step-up 再認証用の WebAuthn assertion challenge を発行する。
// challenge は現在の認証済み session id をキーに保存し、complete で method=webauthn として検証する。
func HandleStepUpWebAuthnChallenge(d httpdeps.Deps, c *echo.Context) error {
	if d.WebAuthnRP == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "webauthn_unavailable", "パスキー認証は利用できません")
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := httpdeps.RequireAuthenticatedAuthn(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	assertion, err := webauthnusecases.BeginWebAuthnAssertion(c.Request().Context(), webauthnhttp.WebAuthnAccountDeps(d), authn.SessionID, authn.UserID)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, assertion)
}

func HandleStartStepUp(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := httpdeps.RequireAuthenticatedAuthn(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	methods, err := authusecases.StepUpStart(c.Request().Context(), stepUpDeps(d), authn.UserID, authn.SessionID)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	out := make([]string, len(methods))
	for i, m := range methods {
		out[i] = string(m)
	}
	return support.NoStoreJSON(c, http.StatusOK, StepUpStartResponse{Methods: out})
}

func HandleCompleteStepUp(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, err := httpdeps.RequireAuthenticatedAuthn(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	var input stepUpCompleteRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if err := authusecases.CompleteStepUp(c.Request().Context(), stepUpDeps(d), authusecases.CompleteStepUpInput{
		Sub:       authn.UserID,
		SessionID: authn.SessionID,
		Method:    authusecases.StepUpMethod(input.Method),
		Password:  input.Password,
		Code:      input.Code,
		Assertion: []byte(input.Assertion),
		Now:       time.Now().UTC(),
	}); err != nil {
		return writeStepUpError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func writeStepUpError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, authusecases.ErrStepUpFailed):
		return support.WriteBrowserError(c, http.StatusForbidden, "step_up_failed", "再認証に失敗しました。入力を確認してください。")
	case errors.Is(err, authusecases.ErrStepUpUnsupportedMethod):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "この再認証方法は利用できません。")
	default:
		return httpdeps.WriteAccountError(c, err)
	}
}
