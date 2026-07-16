// Package usecases implements protocol-neutral claim projection.
package usecases

import (
	"fmt"
	"strconv"
	"strings"

	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
)

type Attributes = claimdomain.Attributes

type ClaimIssuanceResult struct {
	NameIDFormat string
	NameIDValue  string
	Claims       []claimdomain.IssuedClaim
}

const (
	AttrSub               = "sub"
	AttrPreferredUsername = "preferred_username"
	AttrEmail             = "email"
	AttrEmailVerified     = "email_verified"
	AttrName              = "name"
	AttrGivenName         = "given_name"
	AttrFamilyName        = "family_name"
	AttrRoles             = "roles"
)

func IssueClaims(policy claimdomain.ClaimMappingPolicy, attrs Attributes) (ClaimIssuanceResult, error) {
	if strings.TrimSpace(policy.NameID.Format) == "" {
		return ClaimIssuanceResult{}, fmt.Errorf("claim issuance: name_id.format is required")
	}
	if strings.TrimSpace(policy.NameID.SourceAttribute) == "" {
		return ClaimIssuanceResult{}, fmt.Errorf("claim issuance: name_id.source_attribute is required")
	}
	nameID, ok := firstNonEmpty(attrs[policy.NameID.SourceAttribute])
	if !ok {
		return ClaimIssuanceResult{}, fmt.Errorf("claim issuance: NameID source attribute %q has no value", policy.NameID.SourceAttribute)
	}
	result := ClaimIssuanceResult{NameIDFormat: policy.NameID.Format, NameIDValue: nameID}
	for i, rule := range policy.Rules {
		if strings.TrimSpace(rule.ClaimType) == "" {
			return ClaimIssuanceResult{}, fmt.Errorf("claim issuance: rule %d has empty claim_type", i)
		}
		if !rule.Source.Valid() {
			return ClaimIssuanceResult{}, fmt.Errorf("claim issuance: rule %d (%s) has unknown source %q", i, rule.ClaimType, rule.Source)
		}
		values, err := resolveRule(rule, attrs, nameID)
		if err != nil {
			return ClaimIssuanceResult{}, err
		}
		if len(values) == 0 {
			if rule.Required {
				return ClaimIssuanceResult{}, fmt.Errorf("claim issuance: required claim %q could not be resolved", rule.ClaimType)
			}
			continue
		}
		result.Claims = append(result.Claims, claimdomain.IssuedClaim{ClaimType: rule.ClaimType, Values: values})
	}
	return result, nil
}

func ResolveUserAttributes(u idmdomain.User) Attributes {
	attrs := Attributes{}
	put := func(key string, values ...string) {
		filtered := nonEmpty(values)
		if len(filtered) > 0 {
			attrs[key] = filtered
		}
	}
	put(AttrSub, u.ID)
	put(AttrPreferredUsername, u.PreferredUsername)
	put(AttrEmailVerified, strconv.FormatBool(u.EmailVerified))
	put(AttrRoles, u.Roles...)
	if u.Email != nil {
		put(AttrEmail, *u.Email)
	}
	if u.Name != nil {
		put(AttrName, *u.Name)
	}
	if u.GivenName != nil {
		put(AttrGivenName, *u.GivenName)
	}
	if u.FamilyName != nil {
		put(AttrFamilyName, *u.FamilyName)
	}
	for key, value := range u.Attributes {
		if values := attributeValueStrings(value); len(values) > 0 {
			attrs[key] = values
		}
	}
	return attrs
}

func resolveRule(rule claimdomain.ClaimMappingRule, attrs Attributes, nameID string) ([]string, error) {
	switch rule.Source {
	case claimdomain.ClaimSourceUserAttribute:
		if strings.TrimSpace(rule.SourceKey) == "" {
			return nil, fmt.Errorf("claim issuance: claim %q with source user_attribute requires source_key", rule.ClaimType)
		}
		return nonEmpty(attrs[rule.SourceKey]), nil
	case claimdomain.ClaimSourceFixed:
		if v := strings.TrimSpace(rule.FixedValue); v != "" {
			return []string{rule.FixedValue}, nil
		}
		return nil, nil
	case claimdomain.ClaimSourceNameID:
		return []string{nameID}, nil
	default:
		return nil, fmt.Errorf("claim issuance: claim %q has unknown source %q", rule.ClaimType, rule.Source)
	}
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			out = append(out, v)
		}
	}
	return out
}

func firstNonEmpty(values []string) (string, bool) {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v, true
		}
	}
	return "", false
}

func attributeValueStrings(v idmdomain.AttributeValue) []string {
	switch v.Type {
	case idmdomain.AttributeTypeString:
		if v.String != nil && strings.TrimSpace(*v.String) != "" {
			return []string{*v.String}
		}
	case idmdomain.AttributeTypeStringArray:
		return nonEmpty(v.StringArray)
	case idmdomain.AttributeTypeNumber:
		if v.Number != nil {
			return []string{strconv.FormatFloat(*v.Number, 'f', -1, 64)}
		}
	case idmdomain.AttributeTypeBoolean:
		if v.Boolean != nil {
			return []string{strconv.FormatBool(*v.Boolean)}
		}
	case idmdomain.AttributeTypeDate:
		if v.Date != nil && strings.TrimSpace(*v.Date) != "" {
			return []string{*v.Date}
		}
	}
	return nil
}
