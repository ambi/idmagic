// /api/account/v1/data-export — エンドユーザー自身の個人データを JSON で取得する
// (self-service, GDPR 第15条 right of access, wi-21)。本ステージは同期生成で、
// 現状 API から得られる profile と接続済みアプリ (consents) をまとめる。
package handlers_http

import (
	"net/http"
	"time"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type accountDataExport struct {
	ExportedAt time.Time                  `json:"exported_at"`
	Profile    AccountProfileResponse     `json:"profile"`
	Consents   []userports.AccountConsent `json:"consents"`
}

func HandleExportAccountData(d Deps, c *echo.Context) error {
	sub, err := requireAuthenticatedSub(d, c)
	if err != nil {
		return writeAccountError(d, c, err)
	}
	user, defs, err := userusecases.GetUserProfile(c.Request().Context(), accountProfileDeps(d), sub)
	if err != nil {
		return writeAccountError(d, c, err)
	}
	consents, err := d.Consents(c.Request().Context(), sub)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, accountDataExport{
		ExportedAt: time.Now().UTC(),
		Profile:    toAccountProfileResponse(user, defs),
		Consents:   consents,
	})
}
