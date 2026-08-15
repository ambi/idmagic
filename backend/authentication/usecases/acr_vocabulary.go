package usecases

import (
	"slices"
	"strings"
)

const (
	ACRPassword = "urn:idmagic:acr:pwd"
	ACRMFA      = "urn:idmagic:acr:mfa"
)

// mfaAMRValues は acr を urn:idmagic:acr:mfa へ引き上げる AMR 値。RFC 8176 の登録値に
// 加えて、記憶済みの信頼済みデバイスを表す非 IANA 拡張値 tdev を含む (wi-91)。tdev は
// 「要素を提示したのではなく端末が記憶されていた」ことを表すため、毎回 MFA を求める
// ルールはこれを充足として認めない (application の sign-in policy が区別する)。
var mfaAMRValues = []string{"otp", "webauthn", "hwk", "swk", "tdev"}

// IsMfaAMR は AMR 値が第二要素相当かを返す。
func IsMfaAMR(method string) bool { return slices.Contains(mfaAMRValues, method) }

func DeriveACR(amr []string) string {
	if slices.ContainsFunc(amr, IsMfaAMR) {
		return ACRMFA
	}
	return ACRPassword
}

func ACRSatisfies(current, requested string) bool {
	for value := range strings.FieldsSeq(requested) {
		if value == current || current == ACRMFA && value == ACRPassword {
			return true
		}
	}
	return false
}
