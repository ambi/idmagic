// /api/account/mfa/recovery-codes/* — backup recovery code の self-service 生成・失効
// (wi-26 / ADR-087)。生成は平文を一度だけ返し、DB には hash のみ保存する。生成・失効はいずれも
// 高 sensitivity 操作として step-up 再認証 (ADR-043) を要求する。
package http

import (
	"net/http"
	"time"

	authusecases "github.com/ambi/idmagic/internal/authentication/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type recoveryCodesGenerateResponse struct {
	Codes       []string  `json:"codes"`
	GeneratedAt time.Time `json:"generated_at"`
}

func (d Deps) recoveryCodesDeps() authusecases.RecoveryCodesDeps {
	return authusecases.RecoveryCodesDeps{
		UserRepo: d.UserRepo, RecoveryCodeRepo: d.RecoveryCodeRepo, Emit: d.Emit,
	}
}

func (d Deps) handleGenerateRecoveryCodes(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireStepUpSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	if d.RecoveryCodeRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "recovery_unavailable", "リカバリコードは利用できません")
	}
	result, err := authusecases.GenerateRecoveryCodes(c.Request().Context(), d.recoveryCodesDeps(), sub, time.Now().UTC())
	if err != nil {
		return d.writeAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, recoveryCodesGenerateResponse{
		Codes: result.Codes, GeneratedAt: result.GeneratedAt,
	})
}

func (d Deps) handleRevokeRecoveryCodes(c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := d.requireStepUpSub(c)
	if err != nil {
		return d.writeAccountError(c, err)
	}
	if d.RecoveryCodeRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "recovery_unavailable", "リカバリコードは利用できません")
	}
	if err := authusecases.RevokeRecoveryCodes(c.Request().Context(), d.recoveryCodesDeps(), sub, time.Now().UTC()); err != nil {
		return d.writeAccountError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}
