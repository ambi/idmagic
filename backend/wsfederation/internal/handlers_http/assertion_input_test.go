package handlers_http

import (
	"reflect"
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	samltoken "github.com/ambi/idmagic/backend/shared/security/tokens_saml"
	feddomain "github.com/ambi/idmagic/backend/wsfederation/domain"
)

func TestAuthnContextMarksOnlyPasswordAuthentication(t *testing.T) {
	if got := authnContext(feddomain.AuthnPassword); got != samltoken.AuthnContextPassword {
		t.Fatalf("authnContext(password) = %v, want password", got)
	}
	for _, method := range []feddomain.AuthnMethodClass{feddomain.AuthnUnspecified, ""} {
		if got := authnContext(method); got != samltoken.AuthnContextUnspecified {
			t.Fatalf("authnContext(%q) = %v, want unspecified", method, got)
		}
	}
}

func TestAssertionAttributesCarryEveryIssuedClaimInOrder(t *testing.T) {
	claims := []claimdomain.IssuedClaim{
		{ClaimType: "http://schemas.xmlsoap.org/claims/UPN", Values: []string{"alice@contoso.com"}},
		{ClaimType: "http://schemas.xmlsoap.org/claims/Group", Values: []string{"admins", "users"}},
	}
	want := []samltoken.Attribute{
		{Name: "http://schemas.xmlsoap.org/claims/UPN", Values: []string{"alice@contoso.com"}},
		{Name: "http://schemas.xmlsoap.org/claims/Group", Values: []string{"admins", "users"}},
	}
	if got := assertionAttributes(claims); !reflect.DeepEqual(got, want) {
		t.Fatalf("assertionAttributes = %+v, want %+v", got, want)
	}
}
