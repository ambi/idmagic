// Package domain owns protocol-neutral claim release policy and issued claims.
package domain

// ClaimMappingSource identifies the source used to produce a claim value.
type ClaimMappingSource string

const (
	ClaimSourceUserAttribute ClaimMappingSource = "user_attribute"
	ClaimSourceFixed         ClaimMappingSource = "fixed"
	ClaimSourceNameID        ClaimMappingSource = "nameid"
)

func (s ClaimMappingSource) Valid() bool {
	switch s {
	case ClaimSourceUserAttribute, ClaimSourceFixed, ClaimSourceNameID:
		return true
	}
	return false
}

type ClaimMappingRule struct {
	ClaimType  string             `json:"claim_type"`
	Source     ClaimMappingSource `json:"source"`
	SourceKey  string             `json:"source_key,omitempty"`
	FixedValue string             `json:"fixed_value,omitempty"`
	Required   bool               `json:"required,omitempty"`
}

type NameIdConfiguration struct {
	Format          string `json:"format"`
	SourceAttribute string `json:"source_attribute"`
}

type ClaimMappingPolicy struct {
	NameID NameIdConfiguration `json:"name_id"`
	Rules  []ClaimMappingRule  `json:"rules,omitempty"`
}

type IssuedClaim struct {
	ClaimType string   `json:"claim_type"`
	Values    []string `json:"values"`
}

// Attributes is the protocol-neutral, multi-valued identity attribute input.
type Attributes map[string][]string
