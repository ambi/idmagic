package usecases

import (
	"net/url"
	"strings"
	"testing"
)

// decodeCookieValue は parseCookies が値へ適用する復号を写したもの。
// QueryUnescape に失敗したら生の値を採る、という規則そのものを oracle の側に持つ。
func decodeCookieValue(raw string) string {
	if decoded, err := url.QueryUnescape(raw); err == nil {
		return decoded
	}
	return raw
}

// FuzzSessionCookieSelection は Cookie 名の選択規則を表明する。
//
// サブドメイン形式のテナントは __Host- 付きの名前を、パス形式は接頭辞のない名前を使う。
// 両方あるときは host-only の方を採らなければならない (REQ-AUTHENTICATION-035)。
// この選択が解決・失効・参照で食い違うと、サインアウトがセッションを失効させないまま成功する。
func FuzzSessionCookieSelection(f *testing.F) {
	f.Add("sid-1", "sid-2")
	f.Add("", "sid-2")
	f.Add("sid-1", "")
	f.Add("", "")
	f.Add("%2F%2Fevil", "plain")
	f.Add("a=b", "c=d")

	f.Fuzz(func(t *testing.T, hostValue, plainValue string) {
		// ";" は Cookie の区切りなので、埋め込むと別の Cookie になってしまう。
		// 前後の空白はパーサが区切りごとに落とすため、写した oracle と食い違う。
		for _, v := range []string{hostValue, plainValue} {
			if strings.Contains(v, ";") || strings.TrimSpace(v) != v {
				return
			}
		}

		header := "__Host-" + SessionCookie + "=" + hostValue + "; " + SessionCookie + "=" + plainValue

		want := decodeCookieValue(hostValue)
		if want == "" {
			want = decodeCookieValue(plainValue)
		}

		if got := sessionIDFromCookieHeader(header); got != want {
			t.Fatalf("sessionIDFromCookieHeader(%q)=%q, want %q", header, got, want)
		}
	})
}

// FuzzSessionCookieHeader は任意の Cookie ヘッダに対して、送られていない名前から値を
// 取り出さないことを表明する。手書きのヘッダ分割なので、区切りと復号の扱いを探索させる。
func FuzzSessionCookieHeader(f *testing.F) {
	f.Add("")
	f.Add(SessionCookie + "=abc")
	f.Add("__Host-" + SessionCookie + "=abc; " + SessionCookie + "=def")
	f.Add("a=1;;;b=2; =3; c")
	f.Add(SessionCookie + "%3Dabc")
	f.Add("__Host-" + SessionCookie + "=a%3Bb")

	f.Fuzz(func(t *testing.T, header string) {
		got := sessionIDFromCookieHeader(header)
		if got == "" {
			return
		}
		if !strings.Contains(header, SessionCookie) {
			t.Fatalf("resolved session id %q from a header that never names the session cookie: %q", got, header)
		}
	})
}

// TestSessionCookieSelectionTrimsSurroundingSpace は、fuzz target が意図的に除外した
// 前後空白の扱いを表で押さえる。
func TestSessionCookieSelectionTrimsSurroundingSpace(t *testing.T) {
	cases := map[string]string{
		"__Host-" + SessionCookie + "=sid-1 ":      "sid-1",
		" __Host-" + SessionCookie + "=sid-1":      "sid-1",
		"a=1;  __Host-" + SessionCookie + "=sid-1": "sid-1",
		SessionCookie + "= sid-1":                  " sid-1",
	}
	for header, want := range cases {
		if got := sessionIDFromCookieHeader(header); got != want {
			t.Errorf("sessionIDFromCookieHeader(%q)=%q, want %q", header, got, want)
		}
	}
}
