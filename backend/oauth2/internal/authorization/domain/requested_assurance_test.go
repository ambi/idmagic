package domain

import "testing"

//spec:covers REQ-OAUTH2-005, REQ-AUTHENTICATION-027: 認可の要求が MFA の水準を求めるかと、max_age か prompt=login で再認証を求めるかを、要求のパラメーターから導くことを固定する。
func TestRequestedAssurance(t *testing.T) {
	mfa, pwd, login, consent := "urn:idmagic:acr:mfa", "urn:idmagic:acr:pwd", "login consent", "consent"
	zero, fiveMinutes := 0, 300
	for _, tc := range []struct {
		name string
		req  AuthorizationRequest
		want RequestedAssurance
	}{
		{"指定なし", AuthorizationRequest{}, RequestedAssurance{}},
		{"MFA の水準", AuthorizationRequest{ACRValues: &mfa}, RequestedAssurance{MFA: true}},
		{"パスワードの水準", AuthorizationRequest{ACRValues: &pwd}, RequestedAssurance{}},
		{"max_age=0", AuthorizationRequest{ACRValues: &mfa, MaxAge: &zero}, RequestedAssurance{MFA: true, Reauthentication: true}},
		{"max_age=300", AuthorizationRequest{MaxAge: &fiveMinutes}, RequestedAssurance{Reauthentication: true}},
		{"prompt=login", AuthorizationRequest{ACRValues: &mfa, Prompt: &login}, RequestedAssurance{MFA: true, Reauthentication: true}},
		{"prompt=consent", AuthorizationRequest{Prompt: &consent}, RequestedAssurance{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.req.RequestedAssurance(); got != tc.want {
				t.Fatalf("RequestedAssurance() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
