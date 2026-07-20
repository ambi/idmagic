package support_http

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// MetricsMiddleware records HTTP RED (Rate/Errors/Duration) signals for every
// request using the matched route template (c.Path()) as the label, never the
// resolved path or tenant/user/client values. It must be registered outermost
// among the request-scoped middleware — ahead of RecoverMiddleware — so a
// recovered panic's resulting 500 response is still observed: RecoverMiddleware
// converts a panic to a normal c.JSON(500, ...) return before it reaches here,
// but only if this middleware wraps it rather than the other way around.
func MetricsMiddleware(m Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if m == nil {
				return next(c)
			}
			route := c.Path()
			if route == "" {
				route = "unmatched"
			}
			observe := m.BeginHTTPRequest(route, c.Request().Method)
			err := next(c)
			status := http.StatusOK
			if response, ok := c.Response().(*echo.Response); ok && response.Status != 0 {
				status = response.Status
			}
			if err != nil && status < http.StatusBadRequest {
				status = http.StatusInternalServerError
			}
			observe(status)
			return err
		}
	}
}
