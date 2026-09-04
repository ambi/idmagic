package bootstrap

import (
	httpsupport "github.com/ambi/idmagic/backend/shared/http/support_http"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
)

// APIConfig is the startup configuration read only by the idmagic (API)
// process: HTTP listener, hardening, security headers, rate limits, and the
// fields SharedConfig does not cover. Parsed once via LoadAPIConfig using
// the same ConfigLoader as LoadSharedConfig, so a caller that checks
// l.Err() once after both calls sees every problem from the whole startup
// attempt together (REQ-SYSTEM-016).
type APIConfig struct {
	Issuer                string
	Addr                  string
	OTelServiceName       string
	LogLevel              string
	RequestIDTrustInbound bool
	TenantBaseDomain      string
	TrustedForwardedHops  int
	DrainGracePeriod      int // seconds

	PaginationCursorSecret Secret

	RateLimits rlports.RateLimitConfigs

	Admission       httpsupport.AdmissionBudget
	Hardening       httpServerHardening
	SecurityHeaders httpsupport.SecurityHeadersConfig
}

// LoadAPIConfig parses every APIConfig field from l, recording every
// missing/malformed value on l (see LoadSharedConfig). It performs no I/O.
func LoadAPIConfig(l *ConfigLoader) APIConfig {
	var cfg APIConfig

	cfg.Issuer = l.URL("ISSUER", "http://localhost:8080")
	cfg.Addr = l.String("ADDR", ":8080")
	cfg.OTelServiceName = l.String("OTEL_SERVICE_NAME", "idmagic")
	cfg.LogLevel = l.Enum("LOG_LEVEL", "info", "debug", "info", "warn", "warning", "error")
	cfg.RequestIDTrustInbound = l.Bool("REQUEST_ID_TRUST_INBOUND", false)
	cfg.TenantBaseDomain = l.String("TENANT_BASE_DOMAIN", "")
	// TRUSTED_FORWARDED_HOPS: a typo here used to silently keep the default
	// (0, trust nothing) instead of failing startup — the motivating example
	// for wi-103's fail-fast requirement.
	cfg.TrustedForwardedHops = l.NonNegativeInt("TRUSTED_FORWARDED_HOPS", 0)
	cfg.DrainGracePeriod = l.NonNegativeInt("DRAIN_GRACE_PERIOD_SECONDS", 5)
	cfg.PaginationCursorSecret = l.Secret("PAGINATION_CURSOR_SECRET")

	cfg.RateLimits = rlports.RateLimitConfigs{
		"token": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_TOKEN_MAX_REQUESTS", 60),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_TOKEN_WINDOW_SECONDS", 60),
		},
		"authorize": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_AUTHORIZE_MAX_REQUESTS", 30),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_AUTHORIZE_WINDOW_SECONDS", 60),
		},
		"par": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_PAR_MAX_REQUESTS", 30),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_PAR_WINDOW_SECONDS", 60),
		},
		"device_authorization": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_DEVICE_AUTHORIZATION_MAX_REQUESTS", 20),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_DEVICE_AUTHORIZATION_WINDOW_SECONDS", 60),
		},
		"backchannel_authentication": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_MAX_REQUESTS", 20),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_BACKCHANNEL_AUTHENTICATION_WINDOW_SECONDS", 60),
		},
		"password_reset": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_PASSWORD_RESET_MAX_REQUESTS", 5),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_PASSWORD_RESET_WINDOW_SECONDS", 900),
		},
		"login": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_LOGIN_MAX_REQUESTS", 20),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_LOGIN_WINDOW_SECONDS", 60),
		},
	}

	cfg.Admission = loadAdmissionBudget(l)
	cfg.Hardening = LoadHTTPServerHardening(l)
	cfg.SecurityHeaders = LoadSecurityHeaders(l)

	return cfg
}

// loadAdmissionBudget reads the per-priority-class concurrency limits the
// admission controller sheds load with (REQ-SYSTEM-018). The defaults are a
// Planning assumption, not a measurement: 256 concurrent requests against the
// default 20-connection pool already means a deep wait for a connection, and
// the management classes are cut at 75% and 50% of that.
//
// The order of the three limits is itself validated. A configuration where
// management_bulk may run more concurrently than interactive_auth inverts the
// degradation order — the single worst way for this mechanism to be wrong — so
// it stops startup rather than falling back to the defaults (REQ-SYSTEM-016).
func loadAdmissionBudget(l *ConfigLoader) httpsupport.AdmissionBudget {
	budget := httpsupport.AdmissionBudget{
		Enabled:             l.Bool("ADMISSION_CONTROL_ENABLED", true),
		MaxConcurrent:       l.PositiveInt("ADMISSION_MAX_CONCURRENT_REQUESTS", 256),
		ManagementLimit:     l.PositiveInt("ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS", 192),
		ManagementBulkLimit: l.PositiveInt("ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS", 128),
	}
	l.Require("ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS",
		budget.ManagementLimit <= budget.MaxConcurrent,
		"must not exceed ADMISSION_MAX_CONCURRENT_REQUESTS")
	l.Require("ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS",
		budget.ManagementBulkLimit <= budget.ManagementLimit,
		"must not exceed ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS")
	return budget
}
