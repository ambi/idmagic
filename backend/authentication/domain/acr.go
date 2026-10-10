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

func ACRSatisfies(current, requested string) bool {
	for value := range strings.FieldsSeq(requested) {
		if value == current || current == ACRMFA && value == ACRPassword {
			return true
		}
	}
	return false
}
