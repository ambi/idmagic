package support

import (
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/ambi/idmagic/internal/shared/logging"

	"github.com/labstack/echo/v5"
)

// RecoverMiddleware localizes a handler panic to the single request that caused
// it (RequestFaultIsolation objective): it recovers the panic, records it with
// its stack and the request_id through the structured application log, converts
// it to a 500 response, and lets the process keep serving other requests.
//
// http.ErrAbortHandler is re-panicked so net/http's own abort semantics stay
// intact. Client aborts (context.Canceled) travel as returned errors — not
// panics — so they never reach this recover path and keep their existing
// non-server-error classification (ClientAbortLogClassification, ClassifyCancel).
// Only genuine panics are treated as server errors here.
func RecoverMiddleware(logger logging.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = logging.Default()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				if err, ok := r.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(r)
				}
				req := c.Request()
				logger.Error(
					req.Context(), "handler panic recovered",
					"panic", fmt.Sprintf("%v", r),
					"method", req.Method,
					"path", req.URL.Path,
					"stack", string(debug.Stack()),
				)
				err = c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			}()
			return next(c)
		}
	}
}
