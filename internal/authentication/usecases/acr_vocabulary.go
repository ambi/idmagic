package usecases

import (
	"slices"
	"strings"
)

const (
	ACRPassword = "urn:idmagic:acr:pwd"
	ACRMFA      = "urn:idmagic:acr:mfa"
)

var mfaAMRValues = []string{"otp", "webauthn", "hwk", "swk"}

func DeriveACR(amr []string) string {
	for _, method := range amr {
		if slices.Contains(mfaAMRValues, method) {
			return ACRMFA
		}
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
