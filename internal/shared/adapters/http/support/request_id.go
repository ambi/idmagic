package support

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/ambi/idmagic/internal/shared/logging"

	"github.com/labstack/echo/v5"
)

// RequestIDHeader carries the per-request correlation id in and out of the
// service.
const RequestIDHeader = "X-Request-ID"

// maxRequestIDLen bounds an accepted inbound id so an attacker-controlled header
// cannot bloat logs or the response header.
const maxRequestIDLen = 128

// RequestIDMiddleware assigns every request a request_id (RequestFaultIsolation
// objective), writes it to the response header, and stores it in the request
// context so the application logger (ADR-018) attaches it to every line for the
// request. It is registered outermost so downstream middleware — including
// recovery — logs under the same id.
//
// trustInbound governs the trust boundary. When false (secure default), an
// inbound X-Request-ID is ignored and a fresh id is always generated, so a
// directly reachable client cannot spoof or collide correlation ids. When true,
// the operator asserts a trusted edge proxy owns and sanitizes the header
// (REQUEST_ID_TRUST_INBOUND), and a well-formed inbound value is reused so proxy
// and application logs share one id. A reused value is still sanitized as
// defense in depth. Note: trusting inbound is distinct from trusting
// X-Forwarded-For — a proxy may sanitize one and pass the other through — so
// this is a dedicated flag rather than TRUSTED_FORWARDED_HOPS.
func RequestIDMiddleware(trustInbound bool) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			id := ""
			if trustInbound {
				id = sanitizeRequestID(req.Header.Get(RequestIDHeader))
			}
			if id == "" {
				id = newRequestID()
			}
			c.Response().Header().Set(RequestIDHeader, id)
			c.SetRequest(req.WithContext(logging.ContextWithRequestID(req.Context(), id)))
			return next(c)
		}
	}
}

// sanitizeRequestID accepts an inbound id only when it is short and limited to
// unambiguous ASCII (letters, digits, '-', '_', '.'). This blocks header/log
// injection and control characters even from a nominally trusted proxy. An
// unacceptable value yields "" so a fresh id is generated instead.
func sanitizeRequestID(v string) string {
	if v == "" || len(v) > maxRequestIDLen {
		return ""
	}
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return ""
		}
	}
	return v
}

// newRequestID returns a random 128-bit hex correlation id.
func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}
