package support_http

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/labstack/echo/v5"
)

// NoStoreJSON は Cache-Control: no-store を付けて JSON を返す。認証・認可に関わる
// レスポンスが中間キャッシュに残らないようにする共通ヘルパ。
func NoStoreJSON(c *echo.Context, status int, body any) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(status, body)
}

// WriteRateLimited は SCL RateLimitedError (429) を返す。login throttle と
// endpoint rate limiter の両方から呼ばれる共通の 429 応答で、oauth2-bound / browser-JSON の
// どちらの binding でも同じ形にする (OAuthError は 429 の追加フィールドを持てないため
// writeOAuthError 経路は使わない、login throttle の既存パターンを一般化)。
func WriteRateLimited(c *echo.Context, retryAfterSeconds int) error {
	c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	return NoStoreJSON(c, http.StatusTooManyRequests, map[string]any{
		"error": "rate_limited", "retry_after_seconds": retryAfterSeconds,
		// message is the English fallback for callers that don't localize by code;
		// UI screens that do localize (LoginPage etc.) map the "rate_limited"
		// code to a translated message instead.
		"message": "Too many requests. Try again later.",
	})
}

// WriteServerError はサーバー内部エラー (5xx) をロギングし、クライアントに Problem Details エラーレスポンスを返す。
func WriteServerError(c *echo.Context, err error) error {
	logger := logging.Default()
	req := c.Request()
	logger.Error(req.Context(), "internal server error", "error", err.Error(), "method", req.Method, "path", req.URL.Path)
	return WriteProblem(c, http.StatusInternalServerError, "internal_server_error", "Internal server error.")
}

// DecodeJSON はリクエスト body を上限付き (64KiB) かつ未知フィールド拒否で復号する。
func DecodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}
