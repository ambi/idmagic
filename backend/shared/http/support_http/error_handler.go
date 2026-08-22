package support_http

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ambi/idmagic/backend/shared/logging"

	"github.com/labstack/echo/v5"
)

// ErrorHandler wraps echo's default HTTP error handler so a request-scoped
// error that resolves to a 5xx response is logged before the client gets its
// response. echo.DefaultHTTPErrorHandler explicitly does not log errors, and
// RecoverMiddleware only covers panics — a handler that plainly returns a
// non-echo.HTTPError (e.g. a raw DB or dependency error) would otherwise 500
// the client with nothing in the application log to diagnose it from.
type quotaExceeded interface {
	error
	IsQuotaExceeded() bool
	GetResource() string
	GetTenantID() string
}

// fieldLengthViolation は spec.LengthError を import せずに受けるための構造的
// インターフェース。文字列長の上限は解析できた内容に対する業務規則なので、
// 各 context の handler がエラー写像を持つかどうかに関わらず 422 で返す。
type fieldLengthViolation interface {
	error
	IsFieldLengthViolation() bool
}

func ErrorHandler(logger logging.Logger, metrics Metrics) echo.HTTPErrorHandler {
	if logger == nil {
		logger = logging.Default()
	}
	return func(c *echo.Context, err error) {
		// VerifyBrowserRequest は 403 を書き終えてからこのエラーを返す。要求は
		// 完全に応答済みなので、未処理のエラーとして記録し直さない。
		if errors.Is(err, ErrBrowserVerificationFailed) {
			return
		}
		if handled, _ := WriteAccessTokenError(c, err); handled {
			return
		}
		if qErr, ok := errors.AsType[quotaExceeded](err); ok {
			logger.Warn(c.Request().Context(), "tenant resource quota exceeded",
				"tenant_id", qErr.GetTenantID(),
				"resource", qErr.GetResource(),
			)
			if metrics != nil {
				metrics.RecordQuotaExceeded(qErr.GetResource())
			}
			_ = WriteProblem(c, http.StatusUnprocessableEntity, "quota_exceeded", err.Error())
			return
		}
		if lengthErr, ok := errors.AsType[fieldLengthViolation](err); ok {
			_ = WriteProblem(c, http.StatusUnprocessableEntity, "field_length_exceeded", lengthErr.Error())
			return
		}

		code := http.StatusInternalServerError
		if tmp := echo.StatusCode(err); tmp != 0 {
			code = tmp
		}
		if code >= 400 {
			req := c.Request()
			logger.Error(
				req.Context(), "unhandled request error",
				"error", err.Error(),
				"stack", fmt.Sprintf("%+v", err),
				"method", req.Method,
				"path", req.URL.Path,
				"status", code,
			)
		}
		problemFallback(c, err, code)
	}
}

// problemFallback replaces echo.DefaultHTTPErrorHandler so a handler error
// that isn't otherwise classified (not a token/quota error) still gets the
// RFC 9457 envelope instead of echo's built-in `{"message": ...}`
// shape. It mirrors DefaultHTTPErrorHandler's status/message derivation
// (including the already-committed guard) but writes Problem Details.
func problemFallback(c *echo.Context, err error, code int) {
	if r, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil && r != nil && r.Committed {
		return
	}

	detail := http.StatusText(code)
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) && httpErr.Message != "" {
		detail = httpErr.Message
	}

	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(code)
		return
	}
	_ = WriteProblem(c, code, problemCodeForStatus(code), detail)
}

// problemCodeForStatus derives a stable error code slug from the HTTP status
// text (e.g. 404 -> "not_found") for responses that have no more specific
// business error code to report.
func problemCodeForStatus(code int) string {
	text := http.StatusText(code)
	if text == "" {
		return "error"
	}
	return strings.ToLower(strings.ReplaceAll(text, " ", "_"))
}
