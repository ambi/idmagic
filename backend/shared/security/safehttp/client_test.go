package safehttp

import (
	"net"
	"testing"
)

func mustParseIP(t *testing.T, s string) net.IP {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("invalid test IP literal %q", s)
	}
	return ip
}

func TestIsPublicIPRejectsInternalRanges(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "10.0.0.1", "169.254.169.254", "192.168.1.1", "100.64.0.1", "0.0.0.0"} {
		if IsPublicIP(mustParseIP(t, host)) {
			t.Errorf("%s accepted, want rejected", host)
		}
	}
}

func TestIsPublicIPAcceptsPublicAddresses(t *testing.T) {
	for _, host := range []string{"8.8.8.8", "1.1.1.1"} {
		if !IsPublicIP(mustParseIP(t, host)) {
			t.Errorf("%s rejected, want accepted", host)
		}
	}
}

func TestSafeIPsRejectsNonPublicIPLiteralHost(t *testing.T) {
	if _, err := SafeIPs(t.Context(), nil, "127.0.0.1"); err == nil {
		t.Fatal("expected error for loopback IP literal host")
	}
}
