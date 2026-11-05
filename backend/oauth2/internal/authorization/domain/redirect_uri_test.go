package domain

import (
	"slices"
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
