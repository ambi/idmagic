// /api/account/mfa/recovery-codes/* — backup recovery code の self-service 生成・失効
// (wi-26 / ADR-087)。生成は平文を一度だけ返し、DB には hash のみ保存する。生成・失効はいずれも
// 高 sensitivity 操作として step-up 再認証 (ADR-043) を要求する。
package handlers_http

import (
	"net/http"
	"time"

	httpdeps "github.com/ambi/idmagic/backend/authentication/deps_http"
	authusecases "github.com/ambi/idmagic/backend/authentication/recovery/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type recoveryCodesGenerateResponse struct {
	Codes       []string  `json:"codes"`
	GeneratedAt time.Time `json:"generated_at"`
}

func recoveryCodesDeps(d httpdeps.Deps) authusecases.RecoveryCodesDeps {
	return authusecases.RecoveryCodesDeps{
		UserRepo: d.UserRepo, RecoveryCodeRepo: d.RecoveryCodeRepo, Emit: d.Emit,
	}
}

func HandleGenerateRecoveryCodes(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if d.RecoveryCodeRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "recovery_unavailable", "Recovery codes are unavailable.")
	}
	result, err := authusecases.GenerateRecoveryCodes(c.Request().Context(), recoveryCodesDeps(d), sub, time.Now().UTC())
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, recoveryCodesGenerateResponse{
		Codes: result.Codes, GeneratedAt: result.GeneratedAt,
	})
}

func HandleRevokeRecoveryCodes(d httpdeps.Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := httpdeps.RequireStepUpSub(d, c)
	if err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	if d.RecoveryCodeRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "recovery_unavailable", "Recovery codes are unavailable.")
	}
	if err := authusecases.RevokeRecoveryCodes(c.Request().Context(), recoveryCodesDeps(d), sub, time.Now().UTC()); err != nil {
		return httpdeps.WriteAccountError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
