package bootstrap

import (
	httpsupport "github.com/ambi/idmagic/internal/shared/adapters/http/support"
)

// loadSecurityHeaders builds the security response header configuration
// (SecurityResponseHeaders / FrameAncestorsPolicy objectives, ADR-076) from env,
// falling back to production-safe defaults. HSTS is off by default (dev http)
// because the TLS terminator owns it; CSP enforces by default and can be dropped
// to report-only for staged rollout.
func loadSecurityHeaders(getenv func(string) string) httpsupport.SecurityHeadersConfig {
	boolEnv := func(key string) bool { return getenv(key) == "true" }
	return httpsupport.SecurityHeadersConfig{
		ReportOnly:            boolEnv("CSP_REPORT_ONLY"),
		ReportURI:             getenv("CSP_REPORT_URI"),
		HSTSEnabled:           boolEnv("HSTS_ENABLED"),
		HSTSMaxAgeSeconds:     envInt("HSTS_MAX_AGE_SECONDS", 31536000),
		HSTSIncludeSubdomains: getenv("HSTS_INCLUDE_SUBDOMAINS") != "false",
	}
}
