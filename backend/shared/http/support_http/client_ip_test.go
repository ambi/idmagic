package support_http

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

//spec:covers REQ-SYSTEM-032: 直接の接続相手から信頼の範囲をたどり、信頼しない最初の送信元を採る。デフォルトでは転送ヘッダーを読まず、途中の不正な値は接続相手へ戻す。
func TestClientIPExtractor_REQ_SYSTEM_032(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		trusted []string
		remote  string
		xff     []string
		want    string
	}{
		{"未設定は直接の接続相手", nil, "192.0.2.10:1234", []string{"198.51.100.1"}, "192.0.2.10"},
		{"未設定の IPv6", nil, "[2001:db8::10]:1234", []string{"198.51.100.1"}, "2001:db8::10"},
		{"信頼の範囲外の接続相手", []string{"10.0.0.0/8"}, "192.0.2.10:1234", []string{"198.51.100.1"}, "192.0.2.10"},
		{"代理一段で偽装値を無視", []string{"10.0.0.1/32"}, "10.0.0.1:1234", []string{"198.51.100.1, 203.0.113.7"}, "203.0.113.7"},
		{"代理二段", []string{"10.0.0.1/32", "10.0.0.2/32"}, "10.0.0.1:1234", []string{"198.51.100.1, 203.0.113.7, 10.0.0.2"}, "203.0.113.7"},
		{"ループバックを暗黙に信頼しない", []string{"10.0.0.1/32"}, "127.0.0.1:1234", []string{"198.51.100.1"}, "127.0.0.1"},
		{"リンクローカルを暗黙に信頼しない", []string{"10.0.0.1/32"}, "169.254.1.1:1234", []string{"198.51.100.1"}, "169.254.1.1"},
		{"プライベートのクライアントで止まる", []string{"10.0.0.1/32"}, "10.0.0.1:1234", []string{"198.51.100.1, 192.168.1.2"}, "192.168.1.2"},
		{"リンクローカルのクライアントで止まる", []string{"10.0.0.1/32"}, "10.0.0.1:1234", []string{"198.51.100.1, 169.254.1.2"}, "169.254.1.2"},
		{"ループバックのクライアントで止まる", []string{"10.0.0.1/32"}, "10.0.0.1:1234", []string{"198.51.100.1, 127.0.0.2"}, "127.0.0.2"},
		{"途中の不正な値", []string{"10.0.0.1/32"}, "10.0.0.1:1234", []string{"198.51.100.1, invalid"}, "10.0.0.1"},
		{"途中の空の値", []string{"10.0.0.1/32"}, "10.0.0.1:1234", []string{"198.51.100.1, "}, "10.0.0.1"},
		{"ヘッダー欠落", []string{"10.0.0.1/32"}, "10.0.0.1:1234", nil, "10.0.0.1"},
		{"すべて信頼なら左端", []string{"10.0.0.0/24"}, "10.0.0.1:1234", []string{"10.0.0.3, 10.0.0.2"}, "10.0.0.3"},
		{"IPv6 の代理", []string{"2001:db8:1::/64"}, "[2001:db8:1::1]:1234", []string{"2001:db8:2::7, 2001:db8:1::2"}, "2001:db8:2::7"},
		{"複数行の途中の信頼しない送信元", []string{"10.0.0.0/24"}, "10.0.0.1:1234", []string{"198.51.100.1, 203.0.113.7", "10.0.0.2"}, "203.0.113.7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trusted := make([]netip.Prefix, len(tc.trusted))
			for i, cidr := range tc.trusted {
				trusted[i] = netip.MustParsePrefix(cidr)
			}
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.RemoteAddr = tc.remote
			for _, value := range tc.xff {
				req.Header.Add("X-Forwarded-For", value)
			}
			req.Header.Set("X-Real-IP", "198.51.100.99")
			if got := ClientIPExtractor(trusted)(req); got != tc.want {
				t.Fatalf("client IP=%q, want %q", got, tc.want)
			}
		})
	}
}
