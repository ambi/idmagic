package domain

import "slices"

// amr の語彙 (RFC8176-AMR-VOCABULARY)。LoginSession.amr に置ける値はここに挙げたものだけで
// あり、この値は ID トークンとアクセストークンの amr クレームとしてリライングパーティーまで
// 届く。語彙を 1 か所に閉じるのは、要素を足すたびに新しい値が黙って外へ出ていくのを防ぐ
// ためである。
const (
	// AMRPassword はパスワードによる認証 (RFC 8176 の登録値)。
	AMRPassword = "pwd"
	// AMROTP はワンタイムパスワードによる認証 (RFC 8176 の登録値)。TOTP がこれである。
	AMROTP = "otp"
	// AMRWebAuthn は WebAuthn クレデンシャルによる認証 (RFC 8176 の登録値)。
	AMRWebAuthn = "webauthn"
	// AMRHardwareKey はハードウェアに保持された鍵による認証 (RFC 8176 の登録値)。将来用。
	AMRHardwareKey = "hwk"
	// AMRSoftwareKey はソフトウェアに保持された鍵による認証 (RFC 8176 の登録値)。将来用。
	AMRSoftwareKey = "swk"
	// AMRRecoveryCode は復旧コードによる認証。IANA 未登録の本アプリ固有の値である。
	AMRRecoveryCode = "rc"
	// AMRTrustedDevice は記憶済みの信頼済みデバイスによる充足。IANA 未登録の本アプリ固有の
	// 値であり、「要素を提示したのではなく端末が記憶されていた」ことを隠さない。
	AMRTrustedDevice = "tdev"
	// AMRFederated は上流の IdP へ委ねた認証。IANA 未登録の本アプリ固有の値である。
	AMRFederated = "federated"
)

// amrVocabulary は宣言された語彙。宣言順に並べる。
var amrVocabulary = []string{
	AMRPassword, AMROTP, AMRWebAuthn, AMRHardwareKey, AMRSoftwareKey,
	AMRRecoveryCode, AMRTrustedDevice, AMRFederated,
}

// mfaAMRValues は acr を urn:idmagic:acr:mfa へ上げる部分集合。
//
// AMRFederated は入らない。上流の IdP が何を検証したかはブローカーに分からないので、
// mfa を名乗ると強度を偽ることになる。AMRTrustedDevice は入るが、「毎回 MFA」を求める
// SignInRule は別途これを充足として認めない (application の sign-in policy が区別する)。
var mfaAMRValues = []string{
	AMROTP, AMRWebAuthn, AMRHardwareKey, AMRSoftwareKey, AMRRecoveryCode, AMRTrustedDevice,
}

// AMRVocabulary は LoginSession.amr に置ける値を宣言順で返す。
func AMRVocabulary() []string { return slices.Clone(amrVocabulary) }

// MfaAMRValues は acr を mfa へ上げる値を返す。
func MfaAMRValues() []string { return slices.Clone(mfaAMRValues) }

// IsMfaAMR は amr の値が第二要素相当かを返す。
func IsMfaAMR(value string) bool { return slices.Contains(mfaAMRValues, value) }

// UnknownAMRValues は語彙の外の値だけを、渡された順で返す。すべて語彙の内側なら空を返す。
//
// bool ではなく値そのものを返すのは、拒否した側が何を拒否したのかを言えるようにするためで
// ある。「不正な amr」とだけ言う error は、認証要素を足した人にとって手がかりにならない。
func UnknownAMRValues(amr []string) []string {
	var unknown []string
	for _, value := range amr {
		if !slices.Contains(amrVocabulary, value) {
			unknown = append(unknown, value)
		}
	}
	return unknown
}
