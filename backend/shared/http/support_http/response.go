package support_http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/labstack/echo/v5"
)

// NoStoreJSON は Cache-Control: no-store を付けて JSON を返す。認証・認可に関わる
// レスポンスが中間キャッシュに残らないようにする共通ヘルパ。
func NoStoreJSON(c *echo.Context, status int, body any) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(status, body)
}

// WriteBrowserError はブラウザ向け API の {error, message} エラー body を返す。
func WriteBrowserError(c *echo.Context, status int, code, message string) error {
	return NoStoreJSON(c, status, map[string]string{"error": code, "message": kernel.EnglishErrorText(message)})
}

// WriteServerError はサーバー内部エラー (5xx) をロギングし、クライアントに JSON エラーレスポンスを返す。
func WriteServerError(c *echo.Context, err error) error {
	logger := logging.Default()
	req := c.Request()
	logger.Error(req.Context(), "internal server error", "error", err.Error(), "method", req.Method, "path", req.URL.Path)
	return WriteBrowserError(c, http.StatusInternalServerError, "internal_server_error", "内部サーバーエラー")
}

// DecodeJSON はリクエスト body を上限付き (64KiB) かつ未知フィールド拒否で復号する。
func DecodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}
