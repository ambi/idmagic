package domain

import (
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
)

const (
	upnClaim   = "http://schemas.xmlsoap.org/claims/UPN"
	emailClaim = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
	groupClaim = "http://schemas.xmlsoap.org/claims/Group"
	tenantClm  = "https://idmagic/claims/tenant"
	nameIDClm  = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier"
	persistent = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
)

func nameIDOnly() claimdomain.ClaimMappingPolicy {
	return claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistent, SourceAttribute: "object_guid"},
	}
}

func TestIssueClaims_HappyPath(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistent, SourceAttribute: "object_guid"},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: upnClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "upn", Required: true},
			{ClaimType: groupClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "groups"},
			{ClaimType: tenantClm, Source: claimdomain.ClaimSourceFixed, FixedValue: "contoso"},
			{ClaimType: nameIDClm, Source: claimdomain.ClaimSourceNameID},
		},
	}
	attrs := claimdomain.Attributes{
		"object_guid": {"AAECAwQFBgc="},
		"upn":         {"alice@contoso.com"},
		"groups":      {"admins", "users"},
		// 未マップ属性は出力されないことの検証用。
		"phone": {"+1-555-0100"},
	}

	got, err := claimusecases.IssueClaims(policy, attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.NameIDFormat != persistent || got.NameIDValue != "AAECAwQFBgc=" {
		t.Fatalf("NameID = %q/%q, want %q/%q", got.NameIDFormat, got.NameIDValue, persistent, "AAECAwQFBgc=")
	}

	want := map[string][]string{
		upnClaim:   {"alice@contoso.com"},
		groupClaim: {"admins", "users"},
		tenantClm:  {"contoso"},
		nameIDClm:  {"AAECAwQFBgc="},
	}
	if len(got.Claims) != len(want) {
		t.Fatalf("emitted %d claims, want %d: %+v", len(got.Claims), len(want), got.Claims)
	}
	for _, c := range got.Claims {
		w, ok := want[c.ClaimType]
		if !ok {
			t.Fatalf("unexpected claim %q (unmapped attribute leaked?)", c.ClaimType)
		}
		if !equalSlices(c.Values, w) {
			t.Fatalf("claim %q values = %v, want %v", c.ClaimType, c.Values, w)
		}
	}
}

func TestIssueClaims_OptionalMissingIsSkipped(t *testing.T) {
	policy := nameIDOnly()
	policy.Rules = []claimdomain.ClaimMappingRule{
		{ClaimType: emailClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "email"}, // optional, missing
	}
	attrs := claimdomain.Attributes{"object_guid": {"id-1"}}

	got, err := claimusecases.IssueClaims(policy, attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Claims) != 0 {
		t.Fatalf("expected no claims for missing optional source, got %+v", got.Claims)
	}
}

func TestIssueClaims_RequiredMissingIsRejected(t *testing.T) {
	policy := nameIDOnly()
	policy.Rules = []claimdomain.ClaimMappingRule{
		{ClaimType: upnClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "upn", Required: true},
	}
	attrs := claimdomain.Attributes{"object_guid": {"id-1"}} // upn 欠落

	if _, err := claimusecases.IssueClaims(policy, attrs); err == nil {
		t.Fatal("expected error for missing required claim, got nil")
	}
}

func TestIssueClaims_NameIDSourceMissingIsRejected(t *testing.T) {
	attrs := claimdomain.Attributes{"upn": {"alice@contoso.com"}} // object_guid 欠落
	if _, err := claimusecases.IssueClaims(nameIDOnly(), attrs); err == nil {
		t.Fatal("expected error for missing NameID source, got nil")
	}
}

func TestIssueClaims_EmptyValuesTreatedAsMissing(t *testing.T) {
	policy := nameIDOnly()
	policy.Rules = []claimdomain.ClaimMappingRule{
		{ClaimType: upnClaim, Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "upn"},
	}
	attrs := claimdomain.Attributes{
		"object_guid": {"id-1"},
		"upn":         {"  ", ""}, // 空白のみ → 値なし扱い
	}
	got, err := claimusecases.IssueClaims(policy, attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Claims) != 0 {
		t.Fatalf("expected blank values to be dropped, got %+v", got.Claims)
	}
}

func TestIssueClaims_PolicyValidation(t *testing.T) {
	tests := map[string]claimdomain.ClaimMappingPolicy{
		"empty name_id format": {
			NameID: claimdomain.NameIdConfiguration{SourceAttribute: "object_guid"},
		},
		"empty name_id source": {
			NameID: claimdomain.NameIdConfiguration{Format: persistent},
		},
		"empty claim_type": {
			NameID: claimdomain.NameIdConfiguration{Format: persistent, SourceAttribute: "object_guid"},
			Rules:  []claimdomain.ClaimMappingRule{{Source: claimdomain.ClaimSourceFixed, FixedValue: "x"}},
		},
		"unknown source": {
			NameID: claimdomain.NameIdConfiguration{Format: persistent, SourceAttribute: "object_guid"},
			Rules:  []claimdomain.ClaimMappingRule{{ClaimType: upnClaim, Source: "ldap_lookup"}},
		},
		"user_attribute without source_key": {
			NameID: claimdomain.NameIdConfiguration{Format: persistent, SourceAttribute: "object_guid"},
			Rules:  []claimdomain.ClaimMappingRule{{ClaimType: upnClaim, Source: claimdomain.ClaimSourceUserAttribute}},
		},
	}
	attrs := claimdomain.Attributes{"object_guid": {"id-1"}}
	for name, policy := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := claimusecases.IssueClaims(policy, attrs); err == nil {
				t.Fatalf("%s: expected error, got nil", name)
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
