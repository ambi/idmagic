package support

import (
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
func ErrorHandler(logger logging.Logger) echo.HTTPErrorHandler {
	if logger == nil {
		logger = logging.Default()
	}
	fallback := echo.DefaultHTTPErrorHandler(false)
	return func(c *echo.Context, err error) {
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
