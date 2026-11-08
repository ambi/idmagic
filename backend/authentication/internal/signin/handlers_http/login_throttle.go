package handlers_http

import (
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

func (d Deps) emitAuthenticationFailure(c *echo.Context, username, reason string) {
	if d.Emit != nil {
		d.Emit(&authdomain.AuthenticationFailed{
			At: time.Now().UTC(), TenantID: support.RequestTenantID(c), Username: username, Reason: reason,
			IP: c.RealIP(), UserAgent: c.Request().UserAgent(),
		})
	}
}

// writeLoginThrottled renders the login throttle's 429 through the same SCL RateLimitedError
// shape the endpoint rate limiter uses, reconciling what was previously an
// undeclared ad hoc "too_many_requests" body.
func writeLoginThrottled(c *echo.Context, retryAfterSeconds int) error {
	return support.WriteRateLimited(c, retryAfterSeconds)
}
