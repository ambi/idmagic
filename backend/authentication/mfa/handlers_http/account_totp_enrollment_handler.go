// /api/account/mfa/totp/* — エンドユーザー自身の TOTP の self-service 登録・解除
// (wi-21 / ADR-042)。登録は確認コード、解除は有効な TOTP コードによる所持証明に加え、
// 解除は step-up 再認証 (ADR-043) を要求する。
package handlers_http

import (
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type totpEnrollmentStartResponse struct {
	Secret      string `json:"secret"`
	OTPAuthURI  string `json:"otpauth_uri"`
	AccountName string `json:"account_name"`
	Issuer      string `json:"issuer"`
}

type totpEnrollmentConfirmRequest struct {
	Secret string `json:"secret"`
	Code   string `json:"code"`
}

type mfaFactorRemoveRequest struct {
	Code string `json:"code"`
}

func accountMfaDeps(d httpdeps.Deps) authusecases.AccountMfaDeps {
	return authusecases.AccountMfaDeps{
		UserRepo: d.UserRepo, MfaFactorRepo: d.MfaFactorRepo, Emit: d.Emit, Issuer: d.Issuer,
	}
}

func HandleStartTotpEnrollment(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := httpdeps.RequireAuthenticatedSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	start, err := authusecases.StartTOTPEnrollment(c.Request().Context(), accountMfaDeps(d), sub)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, totpEnrollmentStartResponse{
		Secret: start.Secret, OTPAuthURI: start.OTPAuthURI,
		AccountName: start.AccountName, Issuer: start.Issuer,
	})
}

func HandleConfirmTotpEnrollment(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// TOTP factor の確定は認証強度を変更する高 sensitivity 操作。
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	var input totpEnrollmentConfirmRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if err := authusecases.ConfirmTOTPEnrollment(c.Request().Context(), accountMfaDeps(d),
		authusecases.ConfirmTOTPEnrollmentInput{
			Sub: sub, Secret: input.Secret, Code: input.Code, Now: time.Now().UTC(),
		}); err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleRemoveTotpFactor(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	// MFA factor の解除は高 sensitivity 操作。step-up 再認証を要求する (ADR-043)。
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	var input mfaFactorRemoveRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if err := authusecases.RemoveTOTPFactor(c.Request().Context(), accountMfaDeps(d),
		authusecases.RemoveTOTPFactorInput{Sub: sub, Code: input.Code, Now: time.Now().UTC()}); err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
