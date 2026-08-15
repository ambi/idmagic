package domain

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

const testMaxAge = 30 * 24 * time.Hour

func TestNewTrustedDeviceIssuesParsableCookieAndStoresOnlyTheHash(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	device, cookie, err := NewTrustedDevice("tenant-1", "alice", "Chrome / macOS", testMaxAge, now)
	if err != nil {
		t.Fatalf("NewTrustedDevice: %v", err)
	}
	if err := device.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	selector, verifier, ok := ParseCookie(cookie)
	if !ok {
		t.Fatalf("ParseCookie(%q) rejected a freshly issued cookie", cookie)
	}
	if selector != device.Selector {
		t.Fatalf("cookie selector = %q, want %q", selector, device.Selector)
	}
	if device.VerifierHash == verifier {
		t.Fatal("the plaintext verifier must never be stored")
	}
	if !device.VerifierMatches(verifier) {
		t.Fatal("the issued verifier must match the stored hash")
	}
	if got := device.ExpiresAt; !got.Equal(now.Add(testMaxAge)) {
		t.Fatalf("ExpiresAt = %v, want %v", got, now.Add(testMaxAge))
	}
}

func TestParseCookieRejectsMalformedValues(t *testing.T) {
	t.Parallel()

	for _, value := range []string{"", "selector-only", ".verifier", "selector."} {
		if _, _, ok := ParseCookie(value); ok {
			t.Fatalf("ParseCookie(%q) accepted a malformed cookie", value)
		}
	}
}

// REQ-AUTHENTICATION-027: 絶対期限と idle 期限のどちらを過ぎても信頼は成立しない。
func TestTrustedDeviceActiveHonorsAbsoluteAndIdleExpiry(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	device := &TrustedDevice{
		CreatedAt: created, LastUsedAt: created,
		ExpiresAt: created.Add(90 * 24 * time.Hour),
	}

	if !device.Active(created.Add(24 * time.Hour)) {
		t.Fatal("a fresh device within both windows must be active")
	}
	// idle 期限 (30 日) は絶対期限 (90 日) より先に効く。
	if device.Active(created.Add(31 * 24 * time.Hour)) {
		t.Fatal("a device unused past the idle window must not be active")
	}
	device.LastUsedAt = created.Add(89 * 24 * time.Hour)
	if device.Active(created.Add(91 * 24 * time.Hour)) {
		t.Fatal("a recently used device past the absolute expiry must not be active")
	}
}

// 絶対有効期間が idle 期限より短いテナントでは、idle 期限が絶対期限を追い越さない。
func TestTrustedDeviceIdleWindowNeverExceedsAbsoluteLifetime(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	device := &TrustedDevice{
		CreatedAt: created, LastUsedAt: created, ExpiresAt: created.Add(24 * time.Hour),
	}

	if !device.Active(created.Add(23 * time.Hour)) {
		t.Fatal("a device inside a short absolute lifetime must be active")
	}
	if device.Active(created.Add(25 * time.Hour)) {
		t.Fatal("a device past a short absolute lifetime must not be active")
	}
}

func TestRotateReplacesTheVerifierAndAdvancesLastUsed(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	device, cookie, err := NewTrustedDevice("tenant-1", "alice", "", testMaxAge, now)
	if err != nil {
		t.Fatalf("NewTrustedDevice: %v", err)
	}
	_, oldVerifier, _ := ParseCookie(cookie)

	later := now.Add(time.Hour)
	rotated, err := device.Rotate(later)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if device.VerifierMatches(oldVerifier) {
		t.Fatal("the rotated device must reject the previous verifier")
	}
	if !device.VerifierMatches(rotated) {
		t.Fatal("the rotated device must accept the new verifier")
	}
	if !device.LastUsedAt.Equal(later) {
		t.Fatalf("LastUsedAt = %v, want %v", device.LastUsedAt, later)
	}
}

func TestRevokeIsIdempotentAndKeepsTheFirstOutcome(t *testing.T) {
	t.Parallel()
	first := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	device := &TrustedDevice{CreatedAt: first, LastUsedAt: first, ExpiresAt: first.Add(testMaxAge)}

	device.Revoke(spec.TrustedDeviceSelfRevoke, first)
	device.Revoke(spec.TrustedDeviceAdminRevoke, first.Add(time.Hour))

	if device.RevokedAt == nil || !device.RevokedAt.Equal(first) {
		t.Fatalf("RevokedAt = %v, want the first revocation time", device.RevokedAt)
	}
	if device.RevokeReason == nil || *device.RevokeReason != spec.TrustedDeviceSelfRevoke {
		t.Fatalf("RevokeReason = %v, want the first reason", device.RevokeReason)
	}
	if device.Active(first) {
		t.Fatal("a revoked device must never be active")
	}
}
