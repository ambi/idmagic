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

// TestLoadAPIConfigAdmissionDefaults pins the shipped thresholds: the
// mechanism is on by default and the three limits are ordered so the
// degradation order they implement is the one docs/capacity.md states.
func TestLoadAPIConfigAdmissionDefaults(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(nil))
	cfg := LoadAPIConfig(l)
	if err := l.Err(); err != nil {
		t.Fatalf("LoadAPIConfig: %v", err)
	}
	if !cfg.Admission.Enabled {
		t.Error("Admission.Enabled = false, want true: shedding off by default protects nothing in production")
	}
	if cfg.Admission.ManagementBulkLimit >= cfg.Admission.ManagementLimit ||
		cfg.Admission.ManagementLimit >= cfg.Admission.MaxConcurrent {
		t.Fatalf("admission limits = bulk %d, management %d, max %d; want strictly increasing",
			cfg.Admission.ManagementBulkLimit, cfg.Admission.ManagementLimit, cfg.Admission.MaxConcurrent)
	}
}

// TestLoadAPIConfigRejectsInvertedAdmissionThresholds covers REQ-SYSTEM-016
// for the worst way this mechanism can be misconfigured: limits that let
// management_bulk run more concurrently than interactive_auth invert the
// degradation order, so shedding would drop login before the exports it
// exists to drop. Startup stops with an aggregated error naming both keys
// rather than falling back to the defaults.
func TestLoadAPIConfigRejectsInvertedAdmissionThresholds(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{
		"ADMISSION_MAX_CONCURRENT_REQUESTS":                 "10",
		"ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS":      "20",
		"ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS": "30",
	}))
	cfg := LoadAPIConfig(l)
	err := l.Err()
	if err == nil {
		t.Fatal("err = nil, want an aggregated admission threshold error")
	}
	for _, key := range []string{
		"ADMISSION_MANAGEMENT_MAX_CONCURRENT_REQUESTS",
		"ADMISSION_MANAGEMENT_BULK_MAX_CONCURRENT_REQUESTS",
	} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("err = %v, want it to name %s", err, key)
		}
	}
	// The refusal must not silently repair the configuration: the caller sees
	// the rejected values, and it is l.Err() that stops startup.
	if cfg.Admission.ManagementBulkLimit != 30 {
		t.Errorf("ManagementBulkLimit = %d, want the rejected value 30 rather than a silent default",
			cfg.Admission.ManagementBulkLimit)
	}
}

func TestLoadAPIConfigRejectsNonPositiveAdmissionLimit(t *testing.T) {
	t.Parallel()
	l := NewConfigLoader(stubEnv(map[string]string{"ADMISSION_MAX_CONCURRENT_REQUESTS": "0"}))
	LoadAPIConfig(l)
	if err := l.Err(); err == nil || !strings.Contains(err.Error(), "ADMISSION_MAX_CONCURRENT_REQUESTS") {
		t.Fatalf("err=%v, want an ADMISSION_MAX_CONCURRENT_REQUESTS range error", err)
	}
}
