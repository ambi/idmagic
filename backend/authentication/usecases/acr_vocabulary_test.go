package usecases

import "testing"

// RFC8176-AMR-VOCABULARY: acr の導出が amr の語彙から決まることを固定する。
// REQ-AUTHENTICATION-036 の内側の判断でもある。復旧コードは利用者が実際に提示した
// 第二要素なので acr を上げ、federated と pwd は上げない。federated が上がらないのは、
// 上流の IdP が何を検証したかをブローカーが知らないからである。
func TestDeriveACRTreatsRecoveryCodeAsASecondFactor(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		amr  []string
		want string
	}{
		"password only":              {[]string{"pwd"}, ACRPassword},
		"password and recovery code": {[]string{"pwd", "rc"}, ACRMFA},
		"password and TOTP":          {[]string{"pwd", "otp"}, ACRMFA},
		"password and webauthn":      {[]string{"pwd", "webauthn"}, ACRMFA},
		"trusted device":             {[]string{"pwd", "tdev"}, ACRMFA},
		"federated":                  {[]string{"federated"}, ACRPassword},
	} {
		if got := DeriveACR(tc.amr); got != tc.want {
			t.Fatalf("%s: acr=%q, want %q", name, got, tc.want)
		}
	}

	// 導出だけでなく、ポリシー側が読む述語も同じことを言う。
	if !IsMfaAMR("rc") {
		t.Fatal("a recovery code is a second factor the user actually presented")
	}
	if IsMfaAMR("federated") {
		t.Fatal("federated must not satisfy a second factor")
	}
}

// RFC8176-AMR-VOCABULARY: acr の充足判定を固定する。REQ-AUTHENTICATION-036 の締め出しは
// DeriveACR が pwd を返したことではなく、その pwd を ACRSatisfies が mfa の要求に対して
// false と読んだことで起きた。導出だけを観測すると、この述語がどちらへ倒れても気づけない。
// mfa は pwd の要求を満たすが、逆は満たさない。要求は空白区切りで複数を並べられる。
func TestACRSatisfiesTreatsMfaAsStrongerThanPassword(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		current, requested string
		want               bool
	}{
		"mfa satisfies mfa":                  {ACRMFA, ACRMFA, true},
		"mfa satisfies password":             {ACRMFA, ACRPassword, true},
		"password satisfies password":        {ACRPassword, ACRPassword, true},
		"password does not satisfy mfa":      {ACRPassword, ACRMFA, false},
		"either of two requested values":     {ACRMFA, ACRPassword + " " + ACRMFA, true},
		"none of the requested values":       {ACRPassword, "urn:example:other", false},
		"an empty request satisfies nothing": {ACRMFA, "", false},
	} {
		if got := ACRSatisfies(tc.current, tc.requested); got != tc.want {
			t.Fatalf("%s: ACRSatisfies(%q, %q)=%v, want %v", name, tc.current, tc.requested, got, tc.want)
		}
	}
}
