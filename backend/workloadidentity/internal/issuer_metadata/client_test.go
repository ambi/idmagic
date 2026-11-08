package issuer_metadata

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

//spec:covers REQ-WORKLOADIDENTITY-015: パス付き issuer の取得先と完全一致、サイズ、鍵 URL を検査する。
func TestDiscoverIssuerMetadata(t *testing.T) {
	const issuer = "https://issuer.example/tenant/"
	for _, tc := range []struct {
		name, document string
		status         int
		valid          bool
	}{
		{"valid", `{"issuer":"https://issuer.example/tenant/","jwks_uri":"https://keys.example/jwks"}`, 200, true},
		{"size limit", `{"issuer":"https://issuer.example/tenant/","jwks_uri":"https://keys.example/jwks"}` + strings.Repeat(" ", (1<<20)-len(`{"issuer":"https://issuer.example/tenant/","jwks_uri":"https://keys.example/jwks"}`)), 200, true},
		{"issuer mismatch", `{"issuer":"https://issuer.example/tenant","jwks_uri":"https://keys.example/jwks"}`, 200, false},
		{"missing key URL", `{"issuer":"https://issuer.example/tenant/"}`, 200, false},
		{"insecure key URL", `{"issuer":"https://issuer.example/tenant/","jwks_uri":"http://keys.example/jwks"}`, 200, false},
		{"bad JSON", `{`, 200, false},
		{"oversize", `{"issuer":"https://issuer.example/tenant/","jwks_uri":"https://keys.example/jwks"}` + strings.Repeat(" ", (1<<20)+1-len(`{"issuer":"https://issuer.example/tenant/","jwks_uri":"https://keys.example/jwks"}`)), 200, false},
		{"failed upstream", `{}`, 503, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := Client{client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet || r.URL.String() != "https://issuer.example/tenant/.well-known/openid-configuration" {
					t.Fatalf("request=%s %s", r.Method, r.URL)
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.document)), Header: http.Header{}}, nil
			})}}
			uri, err := c.Discover(t.Context(), issuer)
			if tc.valid {
				if err != nil || uri != "https://keys.example/jwks" {
					t.Fatalf("uri=%s err=%v", uri, err)
				}
			} else if err == nil {
				t.Fatalf("accepted invalid metadata: %s", uri)
			}
		})
	}
}

//spec:covers REQ-WORKLOADIDENTITY-015: 危険な初期 URL は通信前に拒否し、実際のクライアントも内部 IP への接続を拒否する。
func TestDiscoverRejectsUnsafeIssuers(t *testing.T) {
	for _, issuer := range []string{"http://issuer.example", "https://user@issuer.example", "https://issuer.example?tenant=a", "https://issuer.example#part", "https://127.0.0.1", "https://[::1]", "https://169.254.169.254", "https://10.0.0.1"} {
		c := NewClient()
		if uri, err := c.Discover(t.Context(), issuer); err == nil {
			t.Fatalf("accepted %s: %s", issuer, uri)
		}
	}
}

//spec:covers REQ-WORKLOADIDENTITY-015: https 以外へのリダイレクトと上限を超えるリダイレクトを、次の要求の前に拒否する。
func TestDiscoverValidatesRedirects(t *testing.T) {
	for _, target := range []string{"http://issuer.example/metadata", "https://issuer.example/metadata"} {
		client := NewClient()
		calls := 0
		client.client.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{target}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		})
		if _, err := client.Discover(t.Context(), "https://issuer.example"); err == nil {
			t.Fatal("accepted unsafe redirect")
		}
		want := 1
		if strings.HasPrefix(target, "https:") {
			want = 3
		}
		if calls != want {
			t.Fatalf("requests=%d want %d", calls, want)
		}
	}
}
