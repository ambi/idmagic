package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"slices"
	"strings"
	"testing"
)

// FuzzVerifyPKCES256 は S256 の往復を表明する。
//
// 正しく導出した challenge は受理し、verifier と challenge のどちらを変異させても拒否する。
// 比較が前方一致や長さ無視へ退行したら、攻撃者は横取りした認可コードを別の verifier で交換できる。
func FuzzVerifyPKCES256(f *testing.F) {
	f.Add("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	f.Add("")
	f.Add(strings.Repeat("a", 128))
	f.Add("パスワード")

	f.Fuzz(func(t *testing.T, verifier string) {
		if len(verifier) > 4096 {
			return
		}
		sum := sha256.Sum256([]byte(verifier))
		challenge := base64.RawURLEncoding.EncodeToString(sum[:])

		if !VerifyPKCES256(verifier, challenge) {
			t.Fatalf("rejected the challenge derived from verifier %q", verifier)
		}
		if VerifyPKCES256(verifier, challenge+"x") {
			t.Fatalf("accepted a challenge with an appended character for verifier %q", verifier)
		}
		if VerifyPKCES256(verifier, challenge[:len(challenge)-1]) {
			t.Fatalf("accepted a truncated challenge for verifier %q", verifier)
		}
		if VerifyPKCES256(verifier, "") {
			t.Fatalf("accepted an empty challenge for verifier %q", verifier)
		}
		if VerifyPKCES256(verifier+"x", challenge) {
			t.Fatalf("accepted a modified verifier against the challenge for %q", verifier)
		}
	})
}

// FuzzParsePromptTokens は、受理する prompt が宣言済みの語だけからなることを表明する。
// none は他の語と併用できない (OIDC Core 3.1.2.1) ので、その組み合わせは必ず拒否される。
func FuzzParsePromptTokens(f *testing.F) {
	f.Add("login")
	f.Add("login consent")
	f.Add("none")
	f.Add("none login")
	f.Add("login login")
	f.Add("")
	f.Add("  \t login  ")
	f.Add("LOGIN")

	f.Fuzz(func(t *testing.T, value string) {
		if len(value) > 4096 {
			return
		}
		tokens, err := ParsePromptTokens(value)
		if err != nil {
			if tokens != (PromptTokens{}) {
				t.Fatalf("ParsePromptTokens returned %+v together with an error", tokens)
			}
			return
		}
		fields := strings.Fields(value)
		for name, set := range map[string]bool{"login": tokens.Login, "consent": tokens.Consent, "none": tokens.None} {
			if set && !slices.Contains(fields, name) {
				t.Fatalf("ParsePromptTokens(%q) set %s without the token appearing in the input", value, name)
			}
		}
		if tokens.None && (tokens.Login || tokens.Consent) {
			t.Fatalf("ParsePromptTokens(%q) combined none with another token", value)
		}
	})
}
