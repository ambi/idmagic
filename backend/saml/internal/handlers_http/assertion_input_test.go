package handlers_http

import (
	"reflect"
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
)

func TestAuthnContextFromAMRMarksOnlyPasswordSessions(t *testing.T) {
	cases := map[string]struct {
		amr  []string
		want samltoken.AuthnContext
	}{
		"password":         {amr: []string{"pwd"}, want: samltoken.AuthnContextPassword},
		"password and otp": {amr: []string{"pwd", "otp"}, want: samltoken.AuthnContextPassword},
		"passkey":          {amr: []string{"hwk"}, want: samltoken.AuthnContextUnspecified},
		"none":             {amr: nil, want: samltoken.AuthnContextUnspecified},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := authnContextFromAMR(tc.amr); got != tc.want {
				t.Fatalf("authnContextFromAMR(%v) = %v, want %v", tc.amr, got, tc.want)
			}
		})
	}
}

func TestAssertionAttributesCarryEveryIssuedClaimInOrder(t *testing.T) {
	claims := []claimdomain.IssuedClaim{
		{ClaimType: "email", Values: []string{"alice@example.com"}},
		{ClaimType: "groups", Values: []string{"admins", "users"}},
	}
	want := []samltoken.Attribute{
		{Name: "email", Values: []string{"alice@example.com"}},
		{Name: "groups", Values: []string{"admins", "users"}},
	}
	if got := assertionAttributes(claims); !reflect.DeepEqual(got, want) {
		t.Fatalf("assertionAttributes = %+v, want %+v", got, want)
	}
}
