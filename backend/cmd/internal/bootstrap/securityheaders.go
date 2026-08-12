package bootstrap

import (
	httpsupport "github.com/ambi/idmagic/backend/shared/http/support_http"
)

// LoadSecurityHeaders builds the security response header configuration
// (SecurityResponseHeaders / FrameAncestorsPolicy objectives) from env,
// falling back to production-safe defaults. HSTS is off by default (dev http)
// because the TLS terminator owns it; CSP enforces by default and can be dropped
// to report-only for staged rollout. A malformed HSTS_MAX_AGE_SECONDS fails
// startup instead of silently keeping the default (wi-103).
func LoadSecurityHeaders(l *ConfigLoader) httpsupport.SecurityHeadersConfig {
	return httpsupport.SecurityHeadersConfig{
		ReportOnly:            l.Bool("CSP_REPORT_ONLY", false),
		ReportURI:             l.String("CSP_REPORT_URI", ""),
		HSTSEnabled:           l.Bool("HSTS_ENABLED", false),
		HSTSMaxAgeSeconds:     l.NonNegativeInt("HSTS_MAX_AGE_SECONDS", 31536000),
		HSTSIncludeSubdomains: l.Bool("HSTS_INCLUDE_SUBDOMAINS", true),
	}
}
