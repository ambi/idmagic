package spec

import "testing"

func TestSaltedHashIsDeterministic(t *testing.T) {
	salt := []byte("tenant-a-salt")
	got1 := SaltedHash(salt, "alice")
	got2 := SaltedHash(salt, "alice")
	if got1 != got2 {
		t.Fatalf("SaltedHash not deterministic: %q != %q", got1, got2)
	}
	if len(got1) != 64 { // hex(sha256) は 32 byte = 64 hex 桁
		t.Fatalf("unexpected hash length %d: %q", len(got1), got1)
	}
}

func TestSaltedHashSeparatesTenants(t *testing.T) {
	a := SaltedHash([]byte("tenant-a-salt"), "alice")
	b := SaltedHash([]byte("tenant-b-salt"), "alice")
	if a == b {
		t.Fatalf("same value hashed identically across tenants: %q", a)
	}
}

func TestSaltedHashSeparatesValues(t *testing.T) {
	salt := []byte("salt")
	if SaltedHash(salt, "alice") == SaltedHash(salt, "bob") {
		t.Fatal("distinct values produced the same hash")
	}
}

func TestTruncateIP(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ipv4 /24", "203.0.113.7", "203.0.113.0/24"},
		{"ipv4 /24 boundary", "203.0.113.255", "203.0.113.0/24"},
		{"ipv6 /48", "2001:db8:1234:5678::1", "2001:db8:1234::/48"},
		{"ipv6 /48 short", "2001:db8:abcd::9", "2001:db8:abcd::/48"},
		{"malformed", "not-an-ip", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := TruncateIP(tc.in); got != tc.want {
				t.Fatalf("TruncateIP(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTruncateIPGroupsSameSubnet(t *testing.T) {
	// 同一 /24 の 2 つの IP は同じ丸め結果になり、相関できる。
	if TruncateIP("198.51.100.10") != TruncateIP("198.51.100.200") {
		t.Fatal("IPs in the same /24 truncated differently")
	}
	// 別 /24 は区別される。
	if TruncateIP("198.51.100.10") == TruncateIP("198.51.101.10") {
		t.Fatal("IPs in different /24 truncated identically")
	}
}
