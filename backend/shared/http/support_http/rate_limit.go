package support_http

import (
	"net/http"
	"strings"
	"time"

	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/labstack/echo/v5"
)

// ExtractClientIP reads the client IP from X-Forwarded-For, trusting only the configured number
// of hops (TrustedForwardedHops) so an attacker cannot spoof around IP-keyed defenses (login
// throttle, endpoint rate limiter) by forging extra hops in front of the trusted proxy chain.
// Returns "" when trustedHops <= 0 (untrusted-by-default) or the header doesn't have enough hops.
func ExtractClientIP(request *http.Request, trustedHops int) string {
	if request == nil || trustedHops <= 0 {
		return ""
	}
	parts := strings.Split(request.Header.Get("X-Forwarded-For"), ",")
	ips := make([]string, 0, len(parts))
	for _, part := range parts {
		if ip := strings.TrimSpace(part); ip != "" {
			ips = append(ips, ip)
		}
	}
	index := len(ips) - 1 - trustedHops
	if index < 0 || index >= len(ips) {
		return ""
	}
	return ips[index]
}

// CheckRateLimit applies the shared endpoint rate limiter for policyID/key, distinct
// from the per-account/per-IP login throttle. When the limiter is nil (not wired — e.g. some
// tests) it allows the request through, matching LoginAttemptThrottle's optional-by-construction
// convention.
//
// Deliberately returns (blocked bool, err error) rather than a single error: WriteRateLimited
// returns nil on a successful write, so a caller written as `if err := CheckRateLimit(...); err
// != nil { return err }` would silently fall through on a blocked-but-successfully-written
// request instead of stopping — the response would already be committed while the handler kept
// running. Callers must check both, mirroring the existing acquireLoginThrottle callers
// (`if err != nil { return err } else if !result.Allowed { return writeLoginThrottled(...) }`):
//
//	if blocked, err := support.CheckRateLimit(c, d.RateLimiter, d.Metrics, "token", key); err != nil {
//		return err
//	} else if blocked {
//		return nil
//	}
//
// metrics may be nil (some tests don't wire it); RecordEndpointRateLimit is skipped in that case.
func CheckRateLimit(c *echo.Context, limiter rlports.RateLimiter, metrics Metrics, policyID, key string) (blocked bool, err error) {
	if limiter == nil {
		return false, nil
	}
	result, err := limiter.Allow(c.Request().Context(), policyID, key, time.Now().UTC())
	if err != nil {
		if metrics != nil {
			metrics.RecordEndpointRateLimit(policyID, "store_unavailable")
		}
		return false, err
	}
	if !result.Allowed {
		if metrics != nil {
			metrics.RecordEndpointRateLimit(policyID, "rate_limited")
		}
		return true, WriteRateLimited(c, result.RetryAfterSeconds)
	}
	if metrics != nil {
		metrics.RecordEndpointRateLimit(policyID, "allowed")
	}
	return false, nil
}
