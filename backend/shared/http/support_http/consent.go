package support_http

import (
	"errors"
	"net/http"

	consentusecases "github.com/ambi/idmagic/backend/oauth2/consent/usecases"

	"github.com/labstack/echo/v5"
)

// WriteConsentError は consent 操作のドメインエラーを HTTP エラーへ変換する。
func (d Deps) WriteConsentError(c *echo.Context, err error) error {
	if errors.Is(err, consentusecases.ErrConsentNotFound) {
		return WriteBrowserError(c, http.StatusNotFound, "consent_not_found", "The consent record does not exist.")
	}
	return err
}
