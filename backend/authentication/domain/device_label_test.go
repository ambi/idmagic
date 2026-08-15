package domain

import "testing"

func TestDeviceLabelKeepsOnlyBrowserAndOSFamily(t *testing.T) {
	t.Parallel()

	cases := []struct{ userAgent, want string }{
		{
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
			"Chrome / macOS",
		},
		{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36 Edg/128.0.0.0",
			"Edge / Windows",
		},
		{"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Version/17.0 Safari/604.1", "Safari / iOS"},
		{"Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0", "Firefox / Linux"},
		{"curl/8.4.0", ""},
	}
	for _, tc := range cases {
		if got := DeviceLabel(tc.userAgent); got != tc.want {
			t.Fatalf("DeviceLabel(%q) = %q, want %q", tc.userAgent, got, tc.want)
		}
	}
}
