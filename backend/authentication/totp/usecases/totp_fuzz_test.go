package usecases

import (
	"encoding/base32"
	"testing"
)

// FuzzVerifyTOTP は、TOTP の受理が宣言した時間窓に収まることを表明する。
//
// 窓の内側で導出したコードは受理し、窓のすぐ外で導出したコードと桁を変えたコードは拒否する。
// 窓が広がると総当たりの成功確率がそのぶん上がるため、境界そのものを表明の対象にする。
func FuzzVerifyTOTP(f *testing.F) {
	f.Add(int64(1_700_000_000), 1, 0)
	f.Add(int64(1_700_000_000), 0, 0)
	f.Add(int64(1_700_000_000), 2, 3)
	f.Add(int64(0), 1, -1)

	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("0123456789abcdef01234"))

	f.Fuzz(func(t *testing.T, unixSeconds int64, window, offsetSteps int) {
		if unixSeconds > 1<<40 || window < 0 || window > 8 {
			return
		}
		if offsetSteps < -32 || offsetSteps > 32 {
			return
		}
		// 窓の全ステップが Unix 元期以降に収まる範囲だけを対象にする。元期直後では
		// VerifyTOTP が負の時刻の生成エラーで走査を打ち切り (fail-closed に false)、
		// 窓の意味が変わる。本番の時刻がそこに入ることはない。
		if unixSeconds < int64(window)*TOTPStepSeconds {
			return
		}

		at := unixSeconds + int64(offsetSteps)*TOTPStepSeconds
		if at < 0 {
			return
		}
		code, err := GenerateTOTP(secret, at)
		if err != nil {
			t.Fatalf("GenerateTOTP(%d): %v", at, err)
		}

		accepted := VerifyTOTP(secret, code, unixSeconds, window)
		withinWindow := offsetSteps >= -window && offsetSteps <= window
		if withinWindow && !accepted {
			t.Fatalf("rejected a code derived %d steps away with window %d", offsetSteps, window)
		}
		// 窓の外でも、別のステップが同じ 6 桁を生むことは起こりうるので、拒否は断定しない。
		// 代わりに、桁数と文字種の検査が効いていることを表明する。
		if VerifyTOTP(secret, code+"0", unixSeconds, window) {
			t.Fatalf("accepted a 7 digit code %q", code+"0")
		}
		if VerifyTOTP(secret, code[:len(code)-1], unixSeconds, window) {
			t.Fatalf("accepted a 5 digit code %q", code[:len(code)-1])
		}
		if VerifyTOTP(secret, "abcdef", unixSeconds, window) {
			t.Fatal("accepted a non numeric code")
		}
	})
}

// FuzzGenerateTOTPShape は、生成されるコードが常に 6 桁の数字であることを表明する。
func FuzzGenerateTOTPShape(f *testing.F) {
	f.Add("JBSWY3DPEHPK3PXP", int64(1_700_000_000))
	f.Add("", int64(0))
	f.Add("not-base32!", int64(1))

	f.Fuzz(func(t *testing.T, secret string, unixSeconds int64) {
		if len(secret) > 1024 || unixSeconds < 0 || unixSeconds > 1<<40 {
			return
		}
		code, err := GenerateTOTP(secret, unixSeconds)
		if err != nil {
			if code != "" {
				t.Fatalf("GenerateTOTP returned %q together with an error", code)
			}
			return
		}
		if len(code) != TOTPDigits {
			t.Fatalf("GenerateTOTP returned %q with %d digits, want %d", code, len(code), TOTPDigits)
		}
		for _, r := range code {
			if r < '0' || r > '9' {
				t.Fatalf("GenerateTOTP returned a non numeric character in %q", code)
			}
		}
	})
}
