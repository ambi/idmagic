package handlers_http

import (
	"slices"
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-AUTHENTICATION-040: 第二要素として加わった amr の webauthn が MfaChallengeSucceeded の factorType=webauthn になり、要素の種類でない復旧コードは記録の対象にならないことを固定する。
func TestSecondFactorKindsMapToMfaFactorTypes(t *testing.T) {
	for amr, want := range map[string]spec.MfaFactorType{"otp": spec.MfaFactorTOTP, "webauthn": spec.MfaFactorWebAuthn} {
		if got, ok := mfaFactorForAMR(amr); !ok || got != want {
			t.Errorf("mfaFactorForAMR(%q) = %q, %v; want %q, true", amr, got, ok, want)
		}
	}
	if got, ok := mfaFactorForAMR("rc"); ok {
		t.Errorf("mfaFactorForAMR(rc) = %q, true; want no factor type", got)
	}
	offered := mfaFactorTypes([]string{"totp", "webauthn", "recovery_code"})
	if !slices.Equal(offered, []spec.MfaFactorType{spec.MfaFactorTOTP, spec.MfaFactorWebAuthn}) {
		t.Errorf("mfaFactorTypes() = %v, want [totp webauthn] without the recovery code", offered)
	}
}
