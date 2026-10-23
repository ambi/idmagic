package domain

import (
	"errors"
	"testing"
	"time"
)

func TestRequiresFreshAuth(t *testing.T) {
	now := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		wfresh   string
		authTime time.Time
		want     bool
	}{
		{"absent: no constraint", "", now.Add(-time.Hour), false},
		{"invalid: no constraint", "abc", now.Add(-time.Hour), false},
		{"negative: no constraint", "-5", now.Add(-time.Hour), false},
		{"wfresh=0 within grace: ok", "0", now.Add(-5 * time.Second), false},
		{"wfresh=0 stale session: re-auth", "0", now.Add(-10 * time.Minute), true},
		{"wfresh=10 fresh: ok", "10", now.Add(-5 * time.Minute), false},
		{"wfresh=10 stale: re-auth", "10", now.Add(-15 * time.Minute), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RequiresFreshAuth(tc.wfresh, tc.authTime, now); got != tc.want {
				t.Fatalf("RequiresFreshAuth(%q) = %v, want %v", tc.wfresh, got, tc.want)
			}
		})
	}
}

func TestResolveAuthnMethod(t *testing.T) {
	const (
		multipleAuthn = "http://schemas.microsoft.com/claims/multipleauthn"
		windows       = "urn:federation:authentication:windows"
	)
	plainRP := WsFedRelyingParty{Wtrealm: "urn:rp"}
	entraRP := WsFedRelyingParty{Wtrealm: EntraRealm, EntraProfile: &EntraFederationProfile{}}
	password := PerformedAuthn{AMR: []string{"pwd"}}
	secondFactor := PerformedAuthn{AMR: []string{"pwd", "otp"}, SecondFactor: true}
	for _, tc := range []struct {
		name      string
		wauth     string
		performed PerformedAuthn
		rp        WsFedRelyingParty
		want      AuthnMethodClass
		wantErr   bool
	}{
		{name: "no wauth reflects performed password", performed: password, rp: plainRP, want: AuthnPassword},
		{name: "no wauth, no amr is unspecified", rp: plainRP, want: AuthnUnspecified},
		{name: "password wauth satisfied by password", wauth: "urn:oasis:names:tc:SAML:2.0:ac:classes:Password", performed: password, rp: plainRP, want: AuthnPassword},
		{name: "password wauth for Entra after a second factor states password", wauth: "urn:oasis:names:tc:SAML:2.0:ac:classes:Password", performed: secondFactor, rp: entraRP, want: AuthnPassword},
		{name: "password wauth not satisfied without password", wauth: "urn:oasis:names:tc:SAML:1.0:am:password", rp: plainRP, wantErr: true},
		{name: "integrated windows is fail-closed", wauth: windows, performed: password, rp: plainRP, wantErr: true},
		{name: "integrated windows is fail-closed for Entra after a second factor", wauth: windows, performed: secondFactor, rp: entraRP, wantErr: true},
		{name: "unknown wauth treated as hint", wauth: "urn:example:custom", performed: password, rp: plainRP, want: AuthnPassword},
		{name: "second factor is not reported to a plain RP", performed: secondFactor, rp: plainRP, want: AuthnPassword},
		{name: "second factor is reported to Entra", performed: secondFactor, rp: entraRP, want: AuthnMultipleAuthn},
		{name: "second factor is reported to Entra under an unknown wauth", wauth: "urn:example:custom", performed: secondFactor, rp: entraRP, want: AuthnMultipleAuthn},
		{name: "password only is not reported to Entra as MFA", performed: password, rp: entraRP, want: AuthnPassword},
		{name: "mfa wauth satisfied by a second factor", wauth: multipleAuthn, performed: secondFactor, rp: entraRP, want: AuthnMultipleAuthn},
		{name: "mfa wauth stays a hint for a plain RP", wauth: multipleAuthn, performed: password, rp: plainRP, want: AuthnPassword},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveAuthnMethod(tc.wauth, tc.performed, tc.rp)
			if (err != nil) != tc.wantErr || got != tc.want {
				t.Fatalf("ResolveAuthnMethod(%q, %+v) = (%q, %v), want (%q, error=%v)", tc.wauth, tc.performed, got, err, tc.want, tc.wantErr)
			}
		})
	}
}

func TestResolveAuthnMethodRequiresASecondFactorForEntraMFA(t *testing.T) {
	entraRP := WsFedRelyingParty{Wtrealm: EntraRealm, EntraProfile: &EntraFederationProfile{}}
	// 記憶済みの端末は SecondFactor を立てないので、amr に tdev があっても第二要素を求める。
	trusted := PerformedAuthn{AMR: []string{"pwd", "tdev"}}
	_, err := ResolveAuthnMethod("http://schemas.microsoft.com/claims/multipleauthn", trusted, entraRP)
	if !errors.Is(err, ErrSecondFactorRequired) {
		t.Fatalf("err = %v, want ErrSecondFactorRequired", err)
	}
}
