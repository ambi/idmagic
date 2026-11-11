package bootstrap

import (
	"net/netip"

	httpsupport "github.com/ambi/idmagic/backend/shared/http/support_http"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/ambi/idmagic/backend/wsfederation"
	wsfedports "github.com/ambi/idmagic/backend/wsfederation/ports"
)

// APIConfig is the startup configuration read only by the idmagic (API)
// process: HTTP listener, hardening, security headers, rate limits, and the
// fields SharedConfig does not cover. Parsed once via LoadAPIConfig using
// the same ConfigLoader as LoadSharedConfig, so a caller that checks
// l.Err() once after both calls sees every problem from the whole startup
// attempt together (REQ-SYSTEM-016).
type APIConfig struct {
	Addr                  string
	OTelServiceName       string
	LogLevel              string
	RequestIDTrustInbound bool
	TenantBaseDomain      string
	TrustedProxies        []netip.Prefix
	DrainGracePeriod      int // seconds

	PaginationCursorSecret Secret
	NegotiateConfigFile    string

	// EntraRealmDiscovery は、Entra のドメインの解除の前に Microsoft のレルム探索を呼ぶかを選ぶ。
	// 無効なら nil で、解除は取り込んだ観測記録だけで判定する。
	EntraRealmDiscovery wsfedports.EntraRealmDiscovery

	RateLimits rlports.RateLimitConfigs

	Admission       httpsupport.AdmissionBudget
	Hardening       httpServerHardening
	SecurityHeaders httpsupport.SecurityHeadersConfig
}

// LoadAPIConfig parses every APIConfig field from l, recording every
// missing/malformed value on l (see LoadSharedConfig). It performs no I/O.
func LoadAPIConfig(l *ConfigLoader) APIConfig {
	var cfg APIConfig

	cfg.Addr = l.String("ADDR", ":8080")
	cfg.OTelServiceName = l.String("OTEL_SERVICE_NAME", "idmagic")
	cfg.LogLevel = l.Enum("LOG_LEVEL", "info", "debug", "info", "warn", "warning", "error")
	cfg.RequestIDTrustInbound = l.Bool("REQUEST_ID_TRUST_INBOUND", false)
	cfg.TenantBaseDomain = l.String("TENANT_BASE_DOMAIN", "")
	cfg.TrustedProxies = loadTrustedProxies(l)
	cfg.DrainGracePeriod = l.NonNegativeInt("DRAIN_GRACE_PERIOD_SECONDS", 5)
	cfg.PaginationCursorSecret = l.Secret("PAGINATION_CURSOR_SECRET")
	cfg.NegotiateConfigFile = l.String("NEGOTIATE_CONFIG_FILE", "")
	cfg.EntraRealmDiscovery = loadEntraRealmDiscovery(l)

	cfg.RateLimits = rlports.RateLimitConfigs{
		"resource_access": {
			MaxRequests:   l.PositiveInt("RATE_LIMIT_RESOURCE_ACCESS_MAX_REQUESTS", 100),
			WindowSeconds: l.PositiveInt("RATE_LIMIT_RESOURCE_ACCESS_WINDOW_SECONDS", 1),
		},
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

func loadTrustedProxies(l *ConfigLoader) []netip.Prefix {
	const key = "TRUSTED_PROXY_CIDRS"
	values := l.StringList(key, nil)
	l.constrain(key, "CIDR")
	proxies := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			l.fail(key, "must contain only CIDR prefixes")
			continue
		}
		proxies = append(proxies, prefix)
	}
	return proxies
}

// loadEntraRealmDiscovery は、Entra のドメインの解除の前に問い合わせる Microsoft のレルム探索を組み立てる。
// 通信はしない。URL は https に限り、試験のために loopback の http だけを許す。
func loadEntraRealmDiscovery(l *ConfigLoader) wsfedports.EntraRealmDiscovery {
	mode := l.Enum("ENTRA_REALM_DISCOVERY", "microsoft", "microsoft", "disabled")
	const urlKey = "ENTRA_REALM_DISCOVERY_URL"
	rawURL := l.String(urlKey, wsfederation.DefaultEntraRealmDiscoveryURL)
	if mode != "microsoft" {
		return nil
	}
	discovery, err := wsfederation.NewEntraRealmDiscovery(rawURL)
	if err != nil {
		l.fail(urlKey, "must be an https URL (http only for loopback) without userinfo, query, or fragment")
		return nil
	}
	return discovery
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
