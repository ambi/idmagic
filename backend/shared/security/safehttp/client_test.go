package safehttp

import (
	"net"
	"net/http"
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

func TestIsPublicIPRejectsEntireCGNATRange(t *testing.T) {
	for _, host := range []string{"100.64.0.0", "100.127.255.255"} {
		if IsPublicIP(mustParseIP(t, host)) {
			t.Errorf("%s accepted, want rejected as CGNAT", host)
		}
	}
	if !IsPublicIP(mustParseIP(t, "100.128.0.1")) {
		t.Error("100.128.0.1 rejected, want accepted outside CGNAT range")
	}
}

func TestNewClientDoesNotUseEnvironmentProxy(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.example.com:8443")
	client := NewClient(Config{})
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("Transport.Proxy is configured, want direct connections only")
	}
}

func TestSafeIPsRejectsNonPublicIPLiteralHost(t *testing.T) {
	if _, err := SafeIPs(t.Context(), nil, "127.0.0.1"); err == nil {
		t.Fatal("expected error for loopback IP literal host")
	}
}
