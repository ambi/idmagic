package domain

import "testing"

// FuzzParseCookie は信頼済みデバイス cookie の分割が往復することを表明する。
//
// ok を返したなら FormatCookie で元の値へ戻らなければならない。最初の "." ではなく
// 最後の "." で切る、前後を空白除去する、といった変更を入れると往復が壊れる。
// selector と verifier の対応がずれると、別端末の記録に対して verifier を照合することになる。
func FuzzParseCookie(f *testing.F) {
	f.Add("selector.verifier")
	f.Add("selector.verifier.extra")
	f.Add(".verifier")
	f.Add("selector.")
	f.Add("no-separator")
	f.Add("")
	f.Add(" selector . verifier ")

	f.Fuzz(func(t *testing.T, value string) {
		selector, verifier, ok := ParseCookie(value)
		if !ok {
			if selector != "" || verifier != "" {
				t.Fatalf("ParseCookie(%q) rejected but returned selector=%q verifier=%q", value, selector, verifier)
			}
			return
		}
		if selector == "" || verifier == "" {
			t.Fatalf("ParseCookie(%q) accepted an empty half: selector=%q verifier=%q", value, selector, verifier)
		}
		if rebuilt := FormatCookie(selector, verifier); rebuilt != value {
			t.Fatalf("ParseCookie/FormatCookie did not round-trip: %q -> %q", value, rebuilt)
		}
	})
}

// FuzzHashVerifier は verifier の照合値が入力を持ち出さないことを表明する。
// 保存も照合もこの形だけで行うため、生の verifier がそのまま残ってはならない。
func FuzzHashVerifier(f *testing.F) {
	f.Add("verifier")
	f.Add("")
	f.Add("0123456789abcdef")

	f.Fuzz(func(t *testing.T, verifier string) {
		hashed := HashVerifier(verifier)
		if len(hashed) != 64 {
			t.Fatalf("HashVerifier(%q) returned %d characters, want 64 hex characters", verifier, len(hashed))
		}
		for _, r := range hashed {
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
				t.Fatalf("HashVerifier(%q) returned a non-hex character %q", verifier, r)
			}
		}
		if verifier != "" && hashed == verifier {
			t.Fatalf("HashVerifier returned its input unchanged for %q", verifier)
		}
	})
}
