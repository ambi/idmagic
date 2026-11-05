package domain

import (
	"net/url"
	"slices"
	"strconv"
	"testing"
)

func TestRedirectURIAllowedAcceptsExactMatch(t *testing.T) {
	registered := []string{"https://other.example/cb", "https://client.example/cb"}
	if !RedirectURIAllowed(registered, "https://client.example/cb") {
		t.Fatal("expected an exactly registered redirect_uri to be allowed")
	}
}

func TestRedirectURIAllowedRejectsNearMisses(t *testing.T) {
	registered := []string{"https://client.example/cb"}
	// いずれも登録済み URI を接頭辞に持つが、RFC 6749 3.1.2.3 は厳密一致を要求する。
	nearMisses := []string{
		"https://client.example/cb/",
		"https://client.example/cb/../../evil",
		"https://client.example/cb?code=stolen",
		"https://client.example/cb#fragment",
		"https://client.example/cb.evil.example",
		"https://client.example/cbextra",
		"https://client.example/c",
		"https://Client.example/cb",
		"https://user@client.example/cb",
		"http://client.example/cb",
	}
	for _, presented := range nearMisses {
		if RedirectURIAllowed(registered, presented) {
			t.Errorf("expected %q to be rejected against %q", presented, registered[0])
		}
	}
}

func TestRedirectURIAllowedRejectsEmptyRegistration(t *testing.T) {
	if RedirectURIAllowed(nil, "https://client.example/cb") {
		t.Fatal("expected no registered redirect_uri to allow nothing")
	}
	if RedirectURIAllowed([]string{"https://client.example/cb"}, "") {
		t.Fatal("expected an empty presented redirect_uri to be rejected")
	}
}

// FuzzRedirectURIAllowed は 2 つの性質を表明する。
//
// 厳密性: 受理するのは登録済みのいずれかとバイト単位で完全一致するときに限る。接頭辞一致や正規化を伴う
// 一致へ退行したら破れる。
//
// 非空虚性: 空でない登録済み URI をそのまま提示したら必ず受理する。厳密性だけでは「常に false を返す」
// 実装も通ってしまうため、両方を置く。
func FuzzRedirectURIAllowed(f *testing.F) {
	f.Add("https://client.example/cb", "https://client.example/cb")
	f.Add("https://client.example/cb", "https://client.example/cb/")
	f.Add("https://client.example/cb", "https://client.example/cb/../evil")
	f.Add("https://client.example/cb", "https://Client.example/cb")
	f.Add("https://client.example/cb", "https://user@client.example/cb")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, registered, presented string) {
		list := []string{"https://other.example/cb", registered}

		if RedirectURIAllowed(list, presented) && !slices.Contains(list, presented) {
			t.Fatalf("accepted a redirect_uri that is not byte-equal to any registered URI: registered=%q presented=%q",
				registered, presented)
		}
		if registered != "" && !RedirectURIAllowed(list, registered) {
			t.Fatalf("rejected a registered redirect_uri presented verbatim: %q", registered)
		}
	})
}

func TestAuthorizationRedirectURIAllowedIgnoresOnlyTheLoopbackPort(t *testing.T) {
	registered := []string{
		"http://127.0.0.1/callback",
		"http://[::1]:8080/cb?app=cli",
		"http://localhost/callback",
		"https://client.example:8443/cb",
	}
	accepted := []string{
		"http://127.0.0.1/callback",
		"http://127.0.0.1:1/callback",
		"http://127.0.0.1:53123/callback",
		"http://127.0.0.1:65535/callback",
		"http://[::1]/cb?app=cli",
		"http://[::1]:8080/cb?app=cli",
		"http://[::1]:49152/cb?app=cli",
		"http://localhost/callback",
		"https://client.example:8443/cb",
	}
	for _, presented := range accepted {
		if !AuthorizationRedirectURIAllowed(registered, presented) {
			t.Errorf("expected %q to be allowed", presented)
		}
	}
	refused := []string{
		"",
		"http://127.0.0.1:0/callback",
		"http://127.0.0.1:65536/callback",
		"http://127.0.0.1:/callback",
		"http://127.0.0.1:+80/callback",
		"http://127.0.0.1:080000/callback",
		"http://127.0.0.1:53123/callback/",
		"http://127.0.0.1:53123/other",
		"http://127.0.0.1:53123/callback?x=1",
		"http://127.0.0.1:53123/callback#x",
		"http://127.0.0.1:53123",
		"http://127.0.0.2:53123/callback",
		"http://[::1]:53123/callback",
		"https://127.0.0.1:53123/callback",
		"HTTP://127.0.0.1:53123/callback",
		"http://127.0.0.1:80@attacker.example/callback",
		"http://user@127.0.0.1:80/callback",
		"http://127.0.0.1.attacker.example/callback",
		"http://127.0.0.1:80\\@attacker.example/callback",
		"http://[::1]:49152/cb?app=cli&x=1",
		"http://[0:0:0:0:0:0:0:1]:8080/cb?app=cli",
		"http://localhost:53123/callback",
		"https://client.example:9443/cb",
		"https://client.example/cb",
	}
	for _, presented := range refused {
		if AuthorizationRedirectURIAllowed(registered, presented) {
			t.Errorf("expected %q to be refused", presented)
		}
	}
}

// 登録値そのものがフラグメントや userinfo を持つ不正な値なら、ポートの例外を適用しない。
// 例外の判定を登録値と提示値の両方へ同じく課さないと、登録の段階で混入した値が
// 一致の基準を広げる。
func TestAuthorizationRedirectURIAllowedRequiresAWellFormedLoopbackRegistration(t *testing.T) {
	cases := map[string]string{
		"http://127.0.0.1/callback#frag":   "http://127.0.0.1:53123/callback#frag",
		"http://127.0.0.1:0/callback":      "http://127.0.0.1:53123/callback",
		"http://127.0.0.1:99999/callback":  "http://127.0.0.1:53123/callback",
		"http://127.0.0.1@evil.example/cb": "http://127.0.0.1:53123@evil.example/cb",
	}
	for registered, presented := range cases {
		if AuthorizationRedirectURIAllowed([]string{registered}, presented) {
			t.Errorf("expected %q to be refused against the malformed registration %q", presented, registered)
		}
	}
}

// FuzzAuthorizationRedirectURIAllowed は 3 つの性質を表明する。
//
// 限定性: 受理した値は登録値と完全一致するか、どちらもループバックの IP の http の値で、
// ポートを取り除いた字句が一致する。字句の分割を誤ってホストやパスを取り違えたら破れる。
// 判定には実装と独立に、url.Parse の結果を使う。
//
// 解析との整合: ポートの違いで受理した値は、url.Parse でも http、同じループバックのホスト、
// userinfo とフラグメントのない URL であり、ポートは 1〜65535 である。
//
// 非空虚性: 空でない登録値をそのまま示したら必ず受理する。
//
//spec:covers RFC9700-REDIRECT-MATCH: 完全一致以外で受理した値は、ループバックの IP の http の登録値とポートだけが異なり、ポートは 1〜65535 である
func FuzzAuthorizationRedirectURIAllowed(f *testing.F) {
	f.Add("http://127.0.0.1/callback", "http://127.0.0.1:53123/callback")
	f.Add("http://[::1]:8080/cb", "http://[::1]/cb")
	f.Add("http://127.0.0.1/cb", "http://127.0.0.1:80@evil.example/cb")
	f.Add("http://127.0.0.1/cb", "http://127.0.0.1.evil.example/cb")
	f.Add("http://127.0.0.1/cb", "http://127.0.0.1:65536/cb")
	f.Add("http://localhost/cb", "http://localhost:1/cb")
	f.Add("https://client.example/cb", "https://client.example/cb")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, registered, presented string) {
		list := []string{"https://other.example/cb", registered}
		if registered != "" && !AuthorizationRedirectURIAllowed(list, registered) {
			t.Fatalf("rejected a registered redirect_uri presented verbatim: %q", registered)
		}
		if !AuthorizationRedirectURIAllowed(list, presented) || slices.Contains(list, presented) {
			return
		}
		got, err := url.Parse(presented)
		if err != nil {
			t.Fatalf("accepted an unparsable redirect_uri %q against %q: %v", presented, registered, err)
		}
		want, err := url.Parse(registered)
		if err != nil {
			t.Fatalf("matched an unparsable registration %q with %q: %v", registered, presented, err)
		}
		host := got.Hostname()
		if got.Scheme != "http" || (host != "127.0.0.1" && host != "::1") || got.User != nil ||
			got.Fragment != "" || got.Opaque != "" {
			t.Fatalf("accepted a non-loopback or malformed redirect_uri %q against %q", presented, registered)
		}
		if want.Scheme != got.Scheme || want.Hostname() != host || want.User != nil ||
			want.EscapedPath() != got.EscapedPath() || want.RawQuery != got.RawQuery ||
			want.ForceQuery != got.ForceQuery {
			t.Fatalf("accepted %q whose parts other than the port differ from %q", presented, registered)
		}
		if port := got.Port(); port != "" {
			if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
				t.Fatalf("accepted %q with the port %q outside 1-65535", presented, port)
			}
		}
	})
}

// ResolveRedirectURI は、要求が配送先を示さないときに登録値で埋めてよいかを決める。
// 埋めてよいのは、登録が一件で、OpenID Connect の認証要求でない場合だけである。
// 登録が複数なら、どれを選んでも利用者の意図しない宛先になり得る。
//
//spec:covers RFC6749-REDIRECT-URI-OPTIONAL: 省いた redirect_uri を一件だけの登録値で埋め、openid か登録が一件でないときは拒否する
func TestResolveRedirectURI(t *testing.T) {
	const registered = "https://client.example/cb"
	single := []string{registered}
	multiple := []string{registered, "https://client.example/other"}

	accepted := []struct {
		name       string
		registered []string
		presented  string
		openID     bool
		want       string
	}{
		{"省いた OAuth の要求は一件の登録値へ届く", single, "", false, registered},
		{"示した値は登録値と一致すればそのまま使う", multiple, "https://client.example/other", true, "https://client.example/other"},
		{"openid の要求も値を示せば受け入れる", single, registered, true, registered},
		{
			"ループバックの値はポートだけ違えば示した値へ届く",
			[]string{"http://127.0.0.1/callback"},
			"http://127.0.0.1:53123/callback", true,
			"http://127.0.0.1:53123/callback",
		},
	}
	for _, tc := range accepted {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveRedirectURI(tc.registered, tc.presented, tc.openID)
			if err != nil || got != tc.want {
				t.Fatalf("ResolveRedirectURI = (%q, %v), want (%q, nil)", got, err, tc.want)
			}
		})
	}

	refused := []struct {
		name       string
		registered []string
		presented  string
		openID     bool
	}{
		{"openid の要求は省けない", single, "", true},
		{"登録が複数なら省けない", multiple, "", false},
		{"登録がなければ省けない", nil, "", false},
		{"未登録の値は拒否する", single, "https://client.example/cb/", false},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveRedirectURI(tc.registered, tc.presented, tc.openID)
			if err == nil {
				t.Fatalf("ResolveRedirectURI = %q, want an error", got)
			}
			if got != "" {
				t.Fatalf("拒否したのに配送先 %q を返した", got)
			}
		})
	}
}
