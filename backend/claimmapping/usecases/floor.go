package usecases

import (
	"fmt"
	"strings"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// coreAttributeKeys are the always-present User fields ResolveUserAttributes populates
// unconditionally. They do not appear in UserAttributeDef and are never gated by
// tenant attribute visibility.
var coreAttributeKeys = map[string]bool{
	AttrUserID:            true,
	AttrPreferredUsername: true,
	AttrEmail:             true,
	AttrEmailVerified:     true,
	AttrName:              true,
	AttrGivenName:         true,
	AttrFamilyName:        true,
	AttrRoles:             true,
}

// reservedClaimTypes are protocol-controlled positions that ClaimMappingRule can never
// target, in any protocol (ADR-151).
var reservedClaimTypes = map[string]bool{
	"iss": true, "sub": true, "aud": true, "exp": true, "iat": true, "nbf": true,
	"jti": true, "azp": true, "nonce": true, "at_hash": true, "c_hash": true,
	"acr": true, "amr": true, "sid": true,
}

// IsReservedClaimType reports whether claimType is an engine-controlled claim position
// that a ClaimMappingRule can never produce or override (ADR-151).
func IsReservedClaimType(claimType string) bool {
	return reservedClaimTypes[strings.ToLower(strings.TrimSpace(claimType))]
}

// IsAttributeReleasable reports whether key may be used as a claim rule or NameID
// source, given the tenant's attribute definitions. Core User fields are always
// releasable. Custom/builtin attributes are releasable unless their visibility is
// Private, or they are not present in defs at all (fail-closed for unknown keys),
// per ADR-151.
func IsAttributeReleasable(key string, defs []userdomain.UserAttributeDef) bool {
	if coreAttributeKeys[key] {
		return true
	}
	for _, d := range defs {
		if d.Key == key {
			return d.Visibility != idmdomain.AttrVisibilityPrivate
		}
	}
	return false
}

// ClaimReleaseDeniedError reports a fail-closed rejection of claim issuance: an
// unresolved required source, a Private-visibility attribute referenced as a rule or
// NameID source, or a reserved claim type targeted by a rule (ADR-151).
type ClaimReleaseDeniedError struct{ Reason string }

func (e *ClaimReleaseDeniedError) Error() string {
	return "claim issuance denied: " + e.Reason
}

// ValidateClaimReleaseRules checks a candidate rule list against the fail-closed floor
// (ADR-151) at write time, so admins get immediate feedback instead of a runtime
// issuance failure. It is the Go implementation of the SCL predicate
// claim_release_rules_within_floor used by UpdateApplicationOidcConfig /
// UpdateApplicationWsFedConfig / UpdateApplicationSamlConfig.
func ValidateClaimReleaseRules(rules []claimdomain.ClaimMappingRule, defs []userdomain.UserAttributeDef) error {
	for _, rule := range rules {
		if IsReservedClaimType(rule.ClaimType) {
			return &ClaimReleaseDeniedError{
				Reason: fmt.Sprintf("claim_type %q is reserved and cannot be produced by a rule", rule.ClaimType),
			}
		}
		if rule.Source == claimdomain.ClaimSourceUserAttribute && !IsAttributeReleasable(rule.SourceKey, defs) {
			return &ClaimReleaseDeniedError{
				Reason: fmt.Sprintf("attribute %q is not releasable (visibility=private or undefined)", rule.SourceKey),
			}
		}
	}
	return nil
}

// IssueClaimsWithFloor enforces the tenant attribute-visibility and reserved-claim-type
// floor before delegating to IssueClaims. It is the single claim resolution path shared
// by OIDC, SAML, and WS-Federation (ADR-151): overrides configured per Application can
// narrow or remap within this floor, but can never reach a Private attribute or an
// engine-controlled claim position.
func IssueClaimsWithFloor(policy claimdomain.ClaimMappingPolicy, attrs Attributes, defs []userdomain.UserAttributeDef) (ClaimIssuanceResult, error) {
	if !IsAttributeReleasable(policy.NameID.SourceAttribute, defs) {
		return ClaimIssuanceResult{}, &ClaimReleaseDeniedError{
			Reason: fmt.Sprintf("NameID source attribute %q is not releasable", policy.NameID.SourceAttribute),
		}
	}
	if err := ValidateClaimReleaseRules(policy.Rules, defs); err != nil {
		return ClaimIssuanceResult{}, err
	}
	result, err := IssueClaims(policy, attrs)
	if err != nil {
		return ClaimIssuanceResult{}, &ClaimReleaseDeniedError{Reason: err.Error()}
	}
	return result, nil
}
