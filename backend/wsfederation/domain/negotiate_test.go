package domain

import (
	"errors"
	"testing"
)

//spec:covers REQ-WSFEDERATION-002: 検証済み Kerberos の認証事実だけが WIA の要求を満たす。
func TestKerberosAMRSatisfiesIntegratedAuthentication(t *testing.T) {
	for wauth := range integratedWauthURIs {
		method, err := ResolveAuthnMethod(wauth, PerformedAuthn{AMR: []string{"wia"}}, WsFedRelyingParty{})
		if err != nil || string(method) != "kerberos" {
			t.Fatalf("wauth=%q method=%q err=%v; want kerberos", wauth, method, err)
		}
		if _, err := ResolveAuthnMethod(wauth, PerformedAuthn{AMR: []string{"pwd"}}, WsFedRelyingParty{}); err == nil {
			t.Fatal("password satisfied integrated authentication")
		}
	}
}

//spec:covers REQ-WSFEDERATION-002: Kerberos と第二要素の認証事実をともに保持し、RP が要求した方式を返す。
func TestKerberosAndEntraMFA(t *testing.T) {
	rp := WsFedRelyingParty{Wtrealm: EntraRealm, EntraProfile: &EntraFederationProfile{}}
	for _, tc := range []struct {
		name         string
		wauth        string
		secondFactor bool
		want         AuthnMethodClass
		wantErr      error
	}{
		{name: "Kerberos alone", want: AuthnKerberos},
		{name: "Kerberos does not imply MFA", wauth: mfaWauthURI, wantErr: ErrSecondFactorRequired},
		{name: "MFA is reported", secondFactor: true, want: AuthnMultipleAuthn},
		{name: "MFA request is satisfied", wauth: mfaWauthURI, secondFactor: true, want: AuthnMultipleAuthn},
		{name: "Windows request preserves Kerberos after MFA", wauth: "urn:federation:authentication:windows", secondFactor: true, want: AuthnKerberos},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveAuthnMethod(tc.wauth, PerformedAuthn{AMR: []string{"wia"}, SecondFactor: tc.secondFactor}, rp)
			if got != tc.want || !errors.Is(err, tc.wantErr) {
				t.Fatalf("method=%q err=%v; want %q, %v", got, err, tc.want, tc.wantErr)
			}
		})
	}
}
