package usecases

import (
	"slices"
	"strings"

	authndomain "github.com/ambi/idmagic/backend/authentication/domain"
)

const (
	ACRPassword = "urn:idmagic:acr:pwd"
	ACRMFA      = "urn:idmagic:acr:mfa"
)

// IsMfaAMR は AMR 値が第二要素相当かを返す。語彙とその部分集合の宣言は
// authentication/domain が 1 か所で持つ (RFC8176-AMR-VOCABULARY)。
func IsMfaAMR(method string) bool { return authndomain.IsMfaAMR(method) }

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
