package client_scim

import (
	"context"
	"testing"
)

func TestSafeIPs_RejectsLoopbackAndPrivateAddresses(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "169.254.1.1", "::1"} {
		if _, err := safeIPs(context.Background(), host); err == nil {
			t.Errorf("safeIPs(%q) should reject a non-public address", host)
		}
	}
}

func TestSafeIPs_AcceptsPublicAddress(t *testing.T) {
	if _, err := safeIPs(context.Background(), "8.8.8.8"); err != nil {
		t.Errorf("safeIPs() should accept a public IP literal, got error: %v", err)
	}
}
