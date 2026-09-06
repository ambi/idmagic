package usecases

import (
	"regexp"
	"testing"
)

const rfc6238SHA1SecretBase32 = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// RFC6238-TOTP: 共有シークレットと時間ステップから OTP を「生成する」側を固定する。
// 入力は RFC 6238 Appendix B の SHA-1 テストベクターそのもので、期待値は RFC が載せている
// 値である。自前の実装の出力を期待値に置くと、時間ステップの計算やダイナミックトランケー
// ションを取り違えたまま自己整合してしまう。
func TestGenerateTOTPRFC6238Vectors(t *testing.T) {
	for _, tc := range []struct {
		at   int64
		want string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	} {
		got, err := GenerateTOTP(rfc6238SHA1SecretBase32, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Fatalf("T=%d: got %s, want %s", tc.at, got, tc.want)
		}
	}
}

// RFC6238-TOTP: 「検証する」側を固定する。時間ステップの許容窓が前後 1 step であり
// 2 step 離れた OTP は受理しないこと、そして受理の可否が共有シークレットに依存すること
// である。窓だけを観測すると、シークレットを見ずに時刻だけから OTP を導く実装が通って
// しまうので、別のシークレットで生成した同時刻の OTP が拒否されることを併せて観測する。
func TestVerifyTOTPWindow(t *testing.T) {
	now := int64(1_700_000_000)
	previous, err := GenerateTOTP(rfc6238SHA1SecretBase32, now-30)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyTOTP(rfc6238SHA1SecretBase32, previous, now, 1) {
		t.Fatal("previous time step should be accepted")
	}
	if VerifyTOTP(rfc6238SHA1SecretBase32, previous, now+30, 1) {
		t.Fatal("code two time steps away should be rejected")
	}

	// 別の共有シークレットの同時刻 OTP は受理しない。
	const otherSecretBase32 = "MFRGGZDFMZTWQ2LKMFRGGZDFMZTWQ2LK"
	other, err := GenerateTOTP(otherSecretBase32, now)
	if err != nil {
		t.Fatal(err)
	}
	if VerifyTOTP(rfc6238SHA1SecretBase32, other, now, 1) {
		t.Fatal("an OTP derived from another shared secret was accepted")
	}
}

func TestGenerateTOTPSecret(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[A-Z2-7]{32}$`).MatchString(secret) {
		t.Fatalf("unexpected secret %q", secret)
	}
}

func TestBuildOTPAuthURI(t *testing.T) {
	uri := BuildOTPAuthURI(rfc6238SHA1SecretBase32, "alice@example.com", "IdMagic")
	for _, part := range []string{
		"otpauth://totp/",
		"secret=" + rfc6238SHA1SecretBase32,
		"issuer=IdMagic",
		"algorithm=SHA1",
		"digits=6",
		"period=30",
	} {
		if !regexp.MustCompile(regexp.QuoteMeta(part)).MatchString(uri) {
			t.Fatalf("URI %q does not contain %q", uri, part)
		}
	}
}
