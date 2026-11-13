package domain

import (
	"slices"
	"strings"
)

const (
	ACRPassword = "urn:idmagic:acr:pwd"
	ACRMFA      = "urn:idmagic:acr:mfa"
)

// DeriveACR は amr から acr を導く。第二要素相当の値がひとつでもあれば mfa へ上がる。
func DeriveACR(amr []string) string {
	if slices.ContainsFunc(amr, IsMfaAMR) {
		return ACRMFA
	}
	return ACRPassword
}

// ACRRequestsMFA は、acr_values の列挙が MFA の水準だけを受け入れるかを返す。パスワードの水準で
// 満たせる列挙と、知らない値だけの列挙は MFA を求めない。
func ACRRequestsMFA(requested string) bool {
	return !ACRSatisfies(ACRPassword, requested) && ACRSatisfies(ACRMFA, requested)
}

func ACRSatisfies(current, requested string) bool {
	for value := range strings.FieldsSeq(requested) {
		if value == current || current == ACRMFA && value == ACRPassword {
			return true
		}
	}
	return false
}
