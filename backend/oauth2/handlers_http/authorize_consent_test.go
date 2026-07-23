package handlers_http

// wi-160 T004.7 RED test for shouldConsumeConsentQuota: the decision of
// whether granting a consent should consume the tenant's consents Hard Quota
// slot (SCL scenario "Hard Quota を超過したリソース作成は拒否される",
// spec/contexts/tenancy.yaml).

import (
	"testing"

	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
)

func TestShouldConsumeConsentQuota(t *testing.T) {
	cases := []struct {
		name     string
		existing *oauthdomain.Consent
		want     bool
	}{
		{"no existing consent", nil, true},
		{"existing revoked consent", &oauthdomain.Consent{State: oauthdomain.ConsentRevoked}, true},
		{"existing granted consent (scope change)", &oauthdomain.Consent{State: oauthdomain.ConsentGranted}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldConsumeConsentQuota(tc.existing); got != tc.want {
				t.Fatalf("shouldConsumeConsentQuota() = %v, want %v", got, tc.want)
			}
		})
	}
}
