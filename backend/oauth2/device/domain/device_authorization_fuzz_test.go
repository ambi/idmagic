package domain

import (
	"strings"
	"testing"
)

// FuzzNormalizeUserCode は user_code の索引キー化が冪等であり、宣言済みの文字集合しか
// 残さないことを表明する。
//
// 正規化がとりこぼすと同じ user_code が別のキーになって照合できず、逆に広く畳むと
// 異なる user_code が同じキーへ潰れて総当たりの空間が縮む。
func FuzzNormalizeUserCode(f *testing.F) {
	f.Add("ABCD-EFGH")
	f.Add("abcd-efgh")
	f.Add("")
	f.Add("A B\tC\nD")
	f.Add("ＡＢＣＤ")
	f.Add("abcdefgh1234")

	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 4096 {
			return
		}
		normalized := NormalizeUserCode(input)

		for _, r := range normalized {
			if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
				t.Fatalf("NormalizeUserCode(%q) kept %q, outside A-Z0-9", input, r)
			}
		}
		if again := NormalizeUserCode(normalized); again != normalized {
			t.Fatalf("NormalizeUserCode is not idempotent: %q -> %q -> %q", input, normalized, again)
		}
		if len(normalized) > len(strings.ToUpper(input)) {
			t.Fatalf("NormalizeUserCode(%q) grew the value to %q", input, normalized)
		}
	})
}
