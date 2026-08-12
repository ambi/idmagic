package bootstrap

import (
	"strings"
	"testing"
)

func TestLoadAPIConfigDefaults(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(nil))
	cfg := LoadAPIConfig(l)
	if err := l.Err(); err != nil {
		t.Fatalf("LoadAPIConfig: %v", err)
	}
	if cfg.Issuer != "http://localhost:8080" {
		t.Errorf("Issuer = %q", cfg.Issuer)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.TrustedForwardedHops != 0 {
		t.Errorf("TrustedForwardedHops = %d, want 0", cfg.TrustedForwardedHops)
	}
	if cfg.RateLimits["login"].MaxRequests != 20 {
		t.Errorf("login MaxRequests = %d, want 20", cfg.RateLimits["login"].MaxRequests)
	}
}

// TestLoadAPIConfigRejectsMalformedTrustedForwardedHops guards the exact
// scenario named in wi-103's motivation: a typo'd TRUSTED_FORWARDED_HOPS
// used to silently keep the default (0, trust nothing) instead of failing
// startup.
func TestLoadAPIConfigRejectsMalformedTrustedForwardedHops(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"TRUSTED_FORWARDED_HOPS": "2x"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "TRUSTED_FORWARDED_HOPS") {
		t.Fatalf("err=%v, want a TRUSTED_FORWARDED_HOPS parse error", err)
	}
}

func TestLoadAPIConfigRejectsNegativeTrustedForwardedHops(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"TRUSTED_FORWARDED_HOPS": "-1"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "TRUSTED_FORWARDED_HOPS") {
		t.Fatalf("err=%v, want a TRUSTED_FORWARDED_HOPS range error", err)
	}
}

func TestLoadAPIConfigRejectsNonAbsoluteIssuer(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"ISSUER": "not-a-url"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "ISSUER") {
		t.Fatalf("err=%v, want an ISSUER absolute-URL error", err)
	}
}

func TestLoadAPIConfigRejectsUnknownLogLevel(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"LOG_LEVEL": "verbose"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Fatalf("err=%v, want a LOG_LEVEL enum error", err)
	}
}

func TestLoadAPIConfigRejectsZeroRateLimitMaxRequests(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"RATE_LIMIT_LOGIN_MAX_REQUESTS": "0"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "RATE_LIMIT_LOGIN_MAX_REQUESTS") {
		t.Fatalf("err=%v, want a RATE_LIMIT_LOGIN_MAX_REQUESTS positive-value error", err)
	}
}

func TestLoadAPIConfigHSTSMaxAgeMalformedFailsFast(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"HSTS_MAX_AGE_SECONDS": "not-a-number"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "HSTS_MAX_AGE_SECONDS") {
		t.Fatalf("err=%v, want an HSTS_MAX_AGE_SECONDS parse error", err)
	}
}
