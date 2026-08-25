package domain

import (
	"net/url"
	"slices"
	"testing"
)

// FuzzParseSignInRequest は passive sign-in の open redirect 防止を表明する。
//
// ValidateSignIn が返す ReplyURL は、必ず RP に登録済みの返信先のいずれかでなければならない。
// wreply を指定した要求が受理された場合は、その wreply とバイト単位で一致していなければならない。
// 接頭辞一致や正規化を伴う一致へ退行すると、攻撃者が RP のドメイン配下に見える別の宛先へ
// トークンを配送できるようになる。
func FuzzParseSignInRequest(f *testing.F) {
	f.Add("wsignin1.0", "https://rp.example", "https://rp.example/cb")
	f.Add("wsignin1.0", "https://rp.example", "https://rp.example/cb/../evil")
	f.Add("wsignin1.0", "https://rp.example", "")
	f.Add("wsignout1.0", "https://rp.example", "https://rp.example/cb")
	f.Add("wsignin1.0", "", "https://attacker.example")
	f.Add("wsignin1.0", "https://RP.example", "https://rp.example/cb")

	f.Fuzz(func(t *testing.T, wa, wtrealm, wreply string) {
		values := url.Values{}
		values.Set("wa", wa)
		values.Set("wtrealm", wtrealm)
		values.Set("wreply", wreply)

		req := ParseSignInRequest(values.Get)

		rp := WsFedRelyingParty{
			Wtrealm:   "https://rp.example",
			ReplyURLs: []string{"https://rp.example/cb", "https://rp.example/other"},
		}
		out, err := ValidateSignIn(req, rp)
		if err != nil {
			if out.ReplyURL != "" || out.Wctx != "" || out.RelyingParty.Wtrealm != "" {
				t.Fatalf("ValidateSignIn returned %+v together with an error", out)
			}
			return
		}
		if !slices.Contains(rp.ReplyURLs, out.ReplyURL) {
			t.Fatalf("resolved an unregistered reply URL %q from wreply=%q", out.ReplyURL, wreply)
		}
		if req.Wreply != "" && out.ReplyURL != req.Wreply {
			t.Fatalf("accepted wreply=%q but resolved to %q", req.Wreply, out.ReplyURL)
		}
		if req.Wtrealm != rp.Wtrealm {
			t.Fatalf("accepted wtrealm=%q against relying party %q", req.Wtrealm, rp.Wtrealm)
		}
	})
}
