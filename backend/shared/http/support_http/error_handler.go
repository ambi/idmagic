package support_http

import (
	"errors"
	"net/http"

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
	IsQuotaExceeded() bool
	GetResource() string
	GetTenantID() string
}

func ErrorHandler(logger logging.Logger, metrics Metrics) echo.HTTPErrorHandler {
	if logger == nil {
		logger = logging.Default()
	}
	fallback := echo.DefaultHTTPErrorHandler(false)
	return func(c *echo.Context, err error) {
		var qErr quotaExceeded
		if errors.As(err, &qErr) {
			logger.Warn(c.Request().Context(), "tenant resource quota exceeded",
				"tenant_id", qErr.GetTenantID(),
				"resource", qErr.GetResource(),
			)
			if metrics != nil {
				metrics.RecordQuotaExceeded(qErr.GetResource())
			}
			fallback(c, echo.NewHTTPError(http.StatusUnprocessableEntity, err.Error()))
			return
		}

		code := http.StatusInternalServerError
		if tmp := echo.StatusCode(err); tmp != 0 {
			code = tmp
		}
		if code >= http.StatusInternalServerError {
			req := c.Request()
			logger.Error(
				req.Context(), "unhandled request error",
				"error", err.Error(),
				"method", req.Method,
				"path", req.URL.Path,
				"status", code,
			)
		}
		fallback(c, err)
	}
}
