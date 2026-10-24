package domain

import (
	"net"
	"net/netip"
	"strings"
	"testing"
)

// netClientIPAllowed は net パッケージで書いた照合で、ClientIPAllowed が従う参照である。
func netClientIPAllowed(clientIP string, cidrs []string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	for _, entry := range cidrs {
		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// FuzzClientIPAllowed は、許可 CIDR の照合が net パッケージの照合と同じ結果を返すことを表明する。
// 照合を誤って広げると、許可していないネットワークからのサインインがポリシーを通る。
func FuzzClientIPAllowed(f *testing.F) {
	seeds := []struct{ ip, cidr string }{
		{"203.0.113.7", "203.0.113.0/24"},
		{"203.0.114.7", "203.0.113.0/24"},
		{" 203.0.113.7 ", "203.0.113.0/24"},
		{"::ffff:203.0.113.7", "203.0.113.0/24"},
		{"203.0.113.7", "::ffff:203.0.113.0/120"},
		{"::ffff:203.0.113.7", "::ffff:203.0.113.0/120"},
		{"2001:db8::1", "2001:db8::/32"},
		{"2001:db9::1", "2001:db8::/32"},
		{"fe80::1%eth0", "fe80::/10"},
		{"::203.0.113.7", "203.0.113.0/24"},
		{"203.0.113.7", "203.0.113.9/24"},
		{"203.0.113.7", "0.0.0.0/0"},
		{"2001:db8::1", "0.0.0.0/0"},
		{"203.0.113.7", "::/0"},
		{"", "203.0.113.0/24"},
		{"not-an-ip", "203.0.113.0/24"},
		{"203.0.113.7", "203.0.113.0/33"},
		{"203.0.113.7", "203.0.113.0"},
		{"203.000.113.7", "203.0.113.0/24"},
	}
	for _, seed := range seeds {
		f.Add(seed.ip, seed.cidr)
	}
	f.Fuzz(func(t *testing.T, ip, cidr string) {
		// 保存されるのは、ポリシーの更新が netip で検証して正規化した CIDR だけである。
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			t.Skip()
		}
		cidrs := []string{prefix.Masked().String()}
		if got, want := ClientIPAllowed(ip, cidrs), netClientIPAllowed(ip, cidrs); got != want {
			t.Fatalf("ClientIPAllowed(%q, %q) = %v, net reference = %v", ip, cidrs[0], got, want)
		}
	})
}

// TestClientIPAllowedAcceptsAddressesInsideTheNetwork は、照合が正当な入力を受理することを表明する。
// 参照との一致だけでは、両方がすべてを拒否する誤りを見逃す。
func TestClientIPAllowedAcceptsAddressesInsideTheNetwork(t *testing.T) {
	for _, tt := range []struct{ ip, cidr string }{
		{"203.0.113.7", "203.0.113.0/24"},
		{"::ffff:203.0.113.7", "203.0.113.0/24"},
		{"2001:db8::1", "2001:db8::/32"},
	} {
		if !ClientIPAllowed(tt.ip, []string{"198.51.100.0/24", tt.cidr}) {
			t.Errorf("ClientIPAllowed(%q, [.., %q]) = false, want true", tt.ip, tt.cidr)
		}
	}
}
