package support

import (
	"errors"
	"net/http"

	oauthusecases "github.com/ambi/idmagic/internal/oauth2/usecases"

	"github.com/labstack/echo/v5"
)

// WriteConsentError は consent 操作のドメインエラーを HTTP エラーへ変換する。
func (d Deps) WriteConsentError(c *echo.Context, err error) error {
	if errors.Is(err, oauthusecases.ErrConsentNotFound) {
		return WriteBrowserError(c, http.StatusNotFound, "consent_not_found", "同意記録が存在しません")
	}
	return err
}
