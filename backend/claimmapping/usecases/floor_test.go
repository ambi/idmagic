package usecases_test

// 主要ユースケース追跡: REQ-CLAIMMAPPING-001。

import (
	"testing"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	claimusecases "github.com/ambi/idmagic/backend/claimmapping/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

const (
	persistentFormat = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
)

func employeeNumberDef() userdomain.UserAttributeDef {
	return userdomain.UserAttributeDef{Key: "employee_number", Visibility: idmdomain.AttrVisibilitySelfReadable}
}

func ssnDef() userdomain.UserAttributeDef {
	return userdomain.UserAttributeDef{Key: "ssn", Visibility: idmdomain.AttrVisibilityPrivate}
}

// TestIsAttributeReleasable_SelfReadableAllowed は wi-73 の動機例 (employeeNumber は
// SelfReadable だが admin が明示的にアプリへ release できる) を検証する (scenario
// 「管理者はApplication単位でclaim releaseを絞り込める」)。
func TestIsAttributeReleasable_SelfReadableAllowed(t *testing.T) {
	defs := []userdomain.UserAttributeDef{employeeNumberDef()}
	if !claimusecases.IsAttributeReleasable("employee_number", defs) {
		t.Fatal("expected SelfReadable attribute to be releasable")
	}
}

func TestIsAttributeReleasable_PrivateRejected(t *testing.T) {
	defs := []userdomain.UserAttributeDef{ssnDef()}
	if claimusecases.IsAttributeReleasable("ssn", defs) {
		t.Fatal("expected Private attribute to be rejected")
	}
}

func TestIsAttributeReleasable_UnknownKeyRejected(t *testing.T) {
	defs := []userdomain.UserAttributeDef{employeeNumberDef()}
	if claimusecases.IsAttributeReleasable("not_defined_anywhere", defs) {
		t.Fatal("expected unknown attribute key to be rejected (fail-closed)")
	}
}

func TestIsAttributeReleasable_CoreAttributesAlwaysAllowed(t *testing.T) {
	for _, key := range []string{
		claimusecases.AttrUserID, claimusecases.AttrEmail, claimusecases.AttrName,
		claimusecases.AttrGivenName, claimusecases.AttrFamilyName,
		claimusecases.AttrPreferredUsername, claimusecases.AttrEmailVerified, claimusecases.AttrRoles,
	} {
		if !claimusecases.IsAttributeReleasable(key, nil) {
			t.Fatalf("expected core attribute %q to always be releasable", key)
		}
	}
}

func TestIsReservedClaimType(t *testing.T) {
	reserved := []string{"iss", "sub", "aud", "exp", "iat", "nbf", "jti", "azp", "nonce", "at_hash", "c_hash", "acr", "amr", "sid"}
	for _, ct := range reserved {
		if !claimusecases.IsReservedClaimType(ct) {
			t.Fatalf("expected %q to be a reserved claim type", ct)
		}
	}
	if claimusecases.IsReservedClaimType("employee_number") {
		t.Fatal("employee_number must not be treated as reserved")
	}
}

func TestIssueClaimsWithFloor_AllowsSelfReadableOverride(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "employee_number", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "employee_number", Required: true},
		},
	}
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}, "employee_number": {"E-123"}}
	defs := []userdomain.UserAttributeDef{employeeNumberDef()}

	got, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Claims) != 1 || got.Claims[0].ClaimType != "employee_number" || got.Claims[0].Values[0] != "E-123" {
		t.Fatalf("unexpected claims: %+v", got.Claims)
	}
}

func TestIssueClaimsWithFloor_RejectsPrivateSourceAttribute(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "ssn_claim", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "ssn"},
		},
	}
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}, "ssn": {"123-45-6789"}}
	defs := []userdomain.UserAttributeDef{ssnDef()}

	if _, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs); err == nil {
		t.Fatal("expected Private attribute source to be rejected fail-closed")
	}
}

func TestIssueClaimsWithFloor_RejectsUnknownSourceAttribute(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "mystery", Source: claimdomain.ClaimSourceUserAttribute, SourceKey: "not_defined_anywhere"},
		},
	}
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}, "not_defined_anywhere": {"leak"}}

	if _, err := claimusecases.IssueClaimsWithFloor(policy, attrs, nil); err == nil {
		t.Fatal("expected unknown attribute source to be rejected fail-closed")
	}
}

func TestIssueClaimsWithFloor_RejectsReservedClaimType(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: claimusecases.AttrUserID},
		Rules: []claimdomain.ClaimMappingRule{
			{ClaimType: "sub", Source: claimdomain.ClaimSourceFixed, FixedValue: "attacker-controlled"},
		},
	}
	attrs := claimdomain.Attributes{claimusecases.AttrUserID: {"user-1"}}

	if _, err := claimusecases.IssueClaimsWithFloor(policy, attrs, nil); err == nil {
		t.Fatal("expected reserved claim_type rule to be rejected")
	}
}

func TestIssueClaimsWithFloor_RejectsPrivateNameIDSource(t *testing.T) {
	policy := claimdomain.ClaimMappingPolicy{
		NameID: claimdomain.NameIdConfiguration{Format: persistentFormat, SourceAttribute: "ssn"},
	}
	attrs := claimdomain.Attributes{"ssn": {"123-45-6789"}}
	defs := []userdomain.UserAttributeDef{ssnDef()}

	if _, err := claimusecases.IssueClaimsWithFloor(policy, attrs, defs); err == nil {
		t.Fatal("expected Private NameID source attribute to be rejected fail-closed")
	}
}
