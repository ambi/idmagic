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
